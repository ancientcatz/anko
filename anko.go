// Package anko implements the core engine for running Tengo rules.
package anko

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/ancientcatz/anko/extras"
	"github.com/d5/tengo/v2"
	"gopkg.in/yaml.v2"
)

// Engine holds the parsed YAML configuration, a structured logger,
// caches compiled Tengo scripts, and a customizable deny list.
type Engine struct {
	Metadata      Metadata
	Env           map[string]any
	Rules         map[string]Rule
	Functions     map[string]string
	compiledCache map[string]*tengo.Compiled
	Logger        *slog.Logger
	denyLibs      []string
	CacheEnabled  bool
	lastInputs    map[string]string
}

// Metadata holds the top‑level anko metadata.
type Metadata struct {
	Name       string   `yaml:"name"`
	Version    string   `yaml:"version"`
	Author     string   `yaml:"author"`
	Language   string   `yaml:"language"`
	Sources    []string `yaml:"sources"`
	Identifier string   `yaml:"identifier"`
}

// NewEngine creates a new Engine with the given *slog.Logger.
// It sets a default deny list.
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		compiledCache: make(map[string]*tengo.Compiled),
		Logger:        logger,
		denyLibs:      []string{},
		CacheEnabled:  true,
		lastInputs:    make(map[string]string),
	}
}

// SetDenyLibs allows customizing the deny list.
func (e *Engine) SetDenyLibs(deny ...string) {
	e.denyLibs = deny
}

// EnableCache turns rule‐level caching on.
func (e *Engine) EnableCache() {
	e.CacheEnabled = true
}

// DisableCache turns rule‐level caching off and clears any existing cache.
func (e *Engine) DisableCache() {
	e.CacheEnabled = false
	e.compiledCache = make(map[string]*tengo.Compiled)
	e.lastInputs = make(map[string]string)
}

// Rule represents an individual rule from the YAML.
type Rule struct {
	Imports []string `yaml:"imports"`
	Code    string   `yaml:"code"`
}

// YAMLData represents the overall YAML structure.
type YAMLData struct {
	Metadata  Metadata          `yaml:"anko"`
	Env       map[string]any    `yaml:"env"`
	Rules     map[string]Rule   `yaml:"rules"`
	Functions map[string]string `yaml:"functions"`
}

// LoadFile loads and parses the YAML file and populates the Engine.
func (e *Engine) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		e.Logger.Error("Error reading YAML file", "error", err)
		return fmt.Errorf("error reading YAML file: %w", err)
	}
	var y YAMLData
	if err := yaml.Unmarshal(data, &y); err != nil {
		e.Logger.Error("Error parsing YAML file", "error", err)
		return fmt.Errorf("error parsing YAML: %w", err)
	}
	e.Metadata = y.Metadata
	e.Env = y.Env
	e.Rules = y.Rules
	e.Functions = y.Functions
	e.Logger.Debug("anko loaded", "filename", filename)
	return nil
}

// RunRule compiles (or reuses a cached) rule and runs it.
// It returns the compiled Tengo script and an error.
func (e *Engine) RunRule(ruleName string) (*tengo.Compiled, error) {
	if e.CacheEnabled {
		if compiledCache, ok := e.compiledCache[ruleName]; ok {
			e.Logger.Info("Running cached rule", "rule", ruleName)
			compiledCache.Run()
			return compiledCache, nil
		}
	}
	rule, exists := e.Rules[ruleName]
	if !exists {
		e.Logger.Error("Rule not found", "rule", ruleName)
		return nil, fmt.Errorf("rule '%s' not found", ruleName)
	}

	preamble, allowedModules := buildPreamble(rule, e.Functions, e.Logger, e.denyLibs)
	finalCode := preamble + "\n" + rule.Code
	e.Logger.Debug("Compiling rule", "rule", ruleName, "code", finalCode)

	script := tengo.NewScript([]byte(finalCode))
	script.SetImports(extras.GetCustomModuleMap(allowedModules, e.Logger))
	script.Add("env", createEnvVariable(e.Env))
	script.Add("url_encode", addURLEncode())
	script.Add("to_title_case", addToTitleCase())

	compiled, err := script.Compile()
	if err != nil {
		e.Logger.Error("Failed to compile rule", "rule", ruleName)
		return nil, fmt.Errorf("failed to compile rule '%s': %w", ruleName, err)
	}
	if e.CacheEnabled {
		e.compiledCache[ruleName] = compiled
	}

	err = compiled.Run()
	if err != nil {
		e.Logger.Error("Engine error", withPrefixes("rule", ruleName, err)...)
		return nil, fmt.Errorf("failed to run rule '%s': %w", ruleName, err)
	}
	return compiled, nil
}

// RunRuleAndGetResult runs a rule and returns the Tengo variable "result".
func (e *Engine) RunRuleAndGetResult(ruleName string) (*tengo.Variable, error) {
	compiled, err := e.RunRule(ruleName)
	if err != nil {
		return nil, err
	}
	resultVar := compiled.Get("result")
	if resultVar == nil {
		e.Logger.Error("Rule did not set 'result'", "rule", ruleName)
		return nil, errors.New("rule did not set the global variable 'result'")
	}
	return resultVar, nil
}

// --- Novel Scraping Rule Functions ---

// SearchRule executes a search rule and validates that each result item meets the schema. THIS COMMENT NEED TO BE UPDATED
func (e *Engine) SearchRule(envVars map[string]any) ([]map[string]any, error) {
	const ruleName = "search"
	if e.CacheEnabled {
		key := serializeEnv(envVars)
		if prev, ok := e.lastInputs[ruleName]; !ok || prev != key {
			delete(e.compiledCache, ruleName)
			e.lastInputs[ruleName] = key
		}
	}
	e.AddEnvVar(ruleName, envVars)
	resultVar, err := e.RunRuleAndGetResult(ruleName)
	if err != nil {
		return nil, err
	}
	arr := resultVar.Array()
	required := []string{"title", "url"}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			e.Logger.Error("SearchRule", "message", "item is not a map", "item", i)
			return nil, fmt.Errorf("SearchRule: item %d is not a map", i)
		}
		for _, key := range required {
			if _, exists := m[key]; !exists {
				e.Logger.Error("SearchRule", "message", "missing required key", "key", key)
				return nil, fmt.Errorf("SearchRule: item %d missing required key: %s", i, key)
			}
		}
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		} else {
			e.Logger.Warn("SearchRule", "message", "skipped non-map item", "item", item)
		}
	}
	return out, nil
}

// NovelInfoRule executes a novel info rule and validates that the result meets the schema. THIS COMMENT NEED TO BE UPDATED
func (e *Engine) NovelInfoRule(envVars map[string]any) (map[string]any, error) {
	const ruleName = "info"
	if e.CacheEnabled {
		key := serializeEnv(envVars)
		if prev, ok := e.lastInputs[ruleName]; !ok || prev != key {
			delete(e.compiledCache, ruleName)
			e.lastInputs[ruleName] = key
		}
	}
	e.AddEnvVar(ruleName, envVars)
	resultVar, err := e.RunRuleAndGetResult(ruleName)
	if err != nil {
		return nil, err
	}
	info := resultVar.Map()
	required := []string{"title", "cover", "author", "description", "status", "genres"}
	for _, key := range required {
		if val, exists := info[key]; !exists {
			return nil, fmt.Errorf("NovelInfoRule: missing required key: %s", key)
		} else if key == "genres" {
			if _, ok := val.([]any); !ok {
				return nil, fmt.Errorf("NovelInfoRule: key 'genres' is not an array")
			}
		}
	}
	return info, nil
}

// ChapterListRule executes a chapter list rule and validates its output. THIS COMMENT NEED TO BE UPDATED
func (e *Engine) ChapterListRule(envVars map[string]any) ([]map[string]any, error) {
	const ruleName = "chapter-list"
	if e.CacheEnabled {
		key := serializeEnv(envVars)
		if prev, ok := e.lastInputs[ruleName]; !ok || prev != key {
			delete(e.compiledCache, ruleName)
			e.lastInputs[ruleName] = key
		}
	}
	e.AddEnvVar("chapter_list", envVars)
	resultVar, err := e.RunRuleAndGetResult(ruleName)
	if err != nil {
		return nil, err
	}
	arr := resultVar.Array()
	required := []string{"title", "url"}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ChapterListRule: item %d is not a map", i)
		}
		for _, key := range required {
			if _, exists := m[key]; !exists {
				return nil, fmt.Errorf("ChapterListRule: item %d missing required key: %s", i, key)
			}
		}
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		} else {
			e.Logger.Warn("ChapterListRule: skipped non-map item", "item", item)
		}
	}
	return out, nil
}

// ContentRule executes a content rule and validates that required keys exist. THIS COMMENT NEED TO BE UPDATED
func (e *Engine) ContentRule(envVars map[string]any) (map[string]any, error) {
	const ruleName = "content"
	if e.CacheEnabled {
		key := serializeEnv(envVars)
		if prev, ok := e.lastInputs[ruleName]; !ok || prev != key {
			delete(e.compiledCache, ruleName)
			e.lastInputs[ruleName] = key
		}
	}
	e.AddEnvVar(ruleName, envVars)
	resultVar, err := e.RunRuleAndGetResult(ruleName)
	if err != nil {
		return nil, err
	}
	content := resultVar.Map()
	required := []string{"title", "content"}
	for _, key := range required {
		if _, exists := content[key]; !exists {
			return nil, fmt.Errorf("ContentRule: missing required key: %s", key)
		}
	}
	return content, nil
}

// GetMetadata returns the metadata loaded from the YAML.
func (e *Engine) GetMetadata() Metadata {
	return e.Metadata
}

// AddEnvVar adds or updates a key-value pair in the Engine's Env map.
// It initializes the Env map if it is nil.
func (e *Engine) AddEnvVar(key string, value any) {
	if e.Env == nil {
		e.Env = make(map[string]any)
	}
	e.Env[key] = value
}
