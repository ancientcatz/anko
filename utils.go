// Package anko contains utility functions.
package anko

import (
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/ancientcatz/anko/extras"
	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// buildPreamble constructs the preamble for a rule using the deny list.
func buildPreamble(rule Rule, functions map[string]string, logger *slog.Logger, denyList []string) (string, []string) {
	var preamble strings.Builder
	var allowedModules []string

	allowedSet := extras.ToSet(stdlib.AllModuleNames()...) // from stdlib
	denySet := extras.ToSet(denyList...)

	for _, imp := range rule.Imports {
		if strings.HasPrefix(imp, "fn:") {
			key := strings.TrimPrefix(imp, "fn:")
			fnLiteral, exists := functions[key]
			if !exists {
				logger.Error("Function not found", "function", key)
				continue
			}
			globalName := "fn_" + strings.ReplaceAll(key, ".", "_")
			preamble.WriteString(fmt.Sprintf("\n%s := %s", globalName, fnLiteral))
		} else {
			if denySet[imp] {
				logger.Warn("Import denied", "import", imp)
				continue
			}
			if allowedSet[imp] || slices.Contains(extras.AllExtraModuleNames(), imp) {
				allowedModules = append(allowedModules, imp)
				preamble.WriteString(fmt.Sprintf("%s := import(\"%s\")\n", imp, imp))
			} else {
				logger.Warn("Unrecognized standard import", "import", imp)
			}
		}
	}
	return preamble.String(), allowedModules
}

// toTengoObject recursively converts a Go value into the corresponding tengo.Object.
func toTengoObject(v any) tengo.Object {
	switch v := v.(type) {
	case string:
		return &tengo.String{Value: v}
	case bool:
		if v {
			return tengo.TrueValue
		}
		return tengo.FalseValue
	case int:
		return &tengo.Int{Value: int64(v)}
	case int8, int16, int32, int64:
		return &tengo.Int{Value: reflect.ValueOf(v).Int()}
	case uint, uint8, uint16, uint32, uint64:
		return &tengo.Int{Value: int64(reflect.ValueOf(v).Uint())}
	case float32, float64:
		return &tengo.Float{Value: reflect.ValueOf(v).Float()}
	case []any:
		arr := make([]tengo.Object, len(v))
		for i, e := range v {
			arr[i] = toTengoObject(e)
		}
		return &tengo.Array{Value: arr}
	case map[string]any:
		mm := make(map[string]tengo.Object, len(v))
		for kk, vv := range v {
			mm[kk] = toTengoObject(vv)
		}
		return &tengo.ImmutableMap{Value: mm}
	default:
		// fallback: string representation
		return &tengo.String{Value: fmt.Sprintf("%v", v)}
	}
}

// createEnvVariable converts the Env map into a Tengo ImmutableMap,
// preserving string, bool, numeric, array, and map types.
func createEnvVariable(envData map[string]any) *tengo.ImmutableMap {
	m := make(map[string]tengo.Object, len(envData))
	for k, v := range envData {
		m[k] = toTengoObject(v)
	}
	return &tengo.ImmutableMap{Value: m}
}

func addURLEncode() *tengo.UserFunction {
	export := &tengo.UserFunction{
		Name: "url_encode",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("url_encode: expected 1 argument")
			}
			str, ok := tengo.ToString(args[0])
			if !ok {
				return nil, fmt.Errorf("url_encode: argument must be a string")
			}
			return &tengo.String{Value: url.QueryEscape(str)}, nil
		},
	}
	return export
}

func addToTitleCase() *tengo.UserFunction {
	export := &tengo.UserFunction{
		Name: "to_title_case",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("to_title_case: expected 1 argument")
			}
			str, ok := tengo.ToString(args[0])
			if !ok {
				return nil, fmt.Errorf("to_title_case: argument must be a string")
			}
			return &tengo.String{Value: toTitleCase(str)}, nil
		},
	}
	return export
}

// serializeEnv turns a map into a reproducible string key.
// You can swap in JSON‑marshal for stable ordering if needed.
func serializeEnv(envVars map[string]any) string {
	return fmt.Sprintf("%#v", envVars)
}

// errorToFields converts an error message into key-value pairs for logging.
func errorToFields(err error) []any {
	s := err.Error()
	parts := strings.SplitN(s, "\n", 2)
	header := parts[0]
	var location string
	if len(parts) > 1 {
		location = strings.TrimSpace(parts[1])
	}

	prefixes := []string{"Runtime Error: ", "Compile Error: ", "Parse Error: "}
	foundPrefix := ""
	for _, prefix := range prefixes {
		if strings.HasPrefix(header, prefix) {
			foundPrefix = prefix
			header = header[len(prefix):]
			break
		}
	}

	var fields []any
	if foundPrefix == "Runtime Error: " && strings.Contains(header, ":") {
		i := strings.Index(header, ":")
		funcName := strings.TrimSpace(header[:i])
		msg := strings.TrimSpace(header[i+1:])
		fields = append(fields, "func", funcName, "message", msg)
	} else {
		fields = append(fields, "message", strings.TrimSpace(header))
	}

	if strings.HasPrefix(location, "at ") {
		location = strings.TrimPrefix(location, "at ")
	}
	if location != "" {
		fields = append(fields, "at", location)
	}

	return fields
}

// withPrefixes prepends context strings to error fields.
func withPrefixes(ctxOne, ctxTwo string, err error) []any {
	fields := []any{ctxOne, ctxTwo}
	fields = append(fields, errorToFields(err)...)
	return fields
}

// the words we want lowercase unless they’re first/last/etc
var lowerCaseWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "the": {},
	"in": {}, "on": {}, "at": {}, "by": {},
	"for": {}, "of": {}, "with": {}, "to": {},
	"but": {}, "or": {}, "nor": {}, "as": {},
}

func hasMultipleCaps(s string) bool {
	cnt := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			cnt++
			if cnt >= 2 {
				return true
			}
		}
	}
	return false
}

// uppercase first rune, leave rest untouched
func capFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toTitleCase(sentence string) string {
	if strings.Contains(sentence, "-") && !strings.Contains(sentence, " ") {
		parts := strings.Split(sentence, "-")
		for i, w := range parts {
			lw := strings.ToLower(w)
			_, small := lowerCaseWords[lw]
			if i == 0 ||
				i == len(parts)-1 ||
				hasMultipleCaps(w) ||
				!small {
				parts[i] = capFirst(w)
			} else {
				parts[i] = lw
			}
		}
		return strings.Join(parts, "-")
	}

	words := strings.Fields(sentence)
	for i, w := range words {
		lw := strings.ToLower(w)
		_, small := lowerCaseWords[lw]

		prev := ""
		if i > 0 {
			prev = words[i-1]
		}

		if i == 0 ||
			i == len(words)-1 ||
			hasMultipleCaps(w) ||
			!small ||
			(prev != "" && (strings.HasSuffix(prev, ":") || strings.HasSuffix(prev, "-"))) {
			words[i] = capFirst(w)
		} else {
			words[i] = lw
		}
	}
	return strings.Join(words, " ")
}
