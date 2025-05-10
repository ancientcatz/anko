// Package extras provides extra Tengo modules.
package extras

import (
	"log/slog"
	"slices"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// ToSet converts a slice of strings into a set.
func ToSet(items ...string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, v := range items {
		set[v] = true
	}
	return set
}

// AllExtraModuleNames returns all extra module names.
func AllExtraModuleNames() []string {
	var names []string
	for name := range ExtraModules {
		names = append(names, name)
	}
	return names
}

// ExtraModules maps extra module names to functions that produce their attribute maps.
var ExtraModules = map[string]func(*slog.Logger) map[string]tengo.Object{
	"log":  logModule,
	"req":  reqModule,
	"html": htmlModule,
	"anko": miscModule,
}

// GetExtraModuleMap creates a ModuleMap for the given extra module names using the provided logger.
func GetExtraModuleMap(logger *slog.Logger, names ...string) *tengo.ModuleMap {
	modules := tengo.NewModuleMap()
	for _, name := range names {
		if fn, ok := ExtraModules[name]; ok {
			modules.AddBuiltinModule(name, fn(logger))
		}
	}
	return modules
}

// GetCustomModuleMap returns a ModuleMap that includes standard modules (from stdlib)
// plus extra modules (only those declared).
func GetCustomModuleMap(allowedModules []string, logger *slog.Logger) *tengo.ModuleMap {
	moduleMap := stdlib.GetModuleMap(allowedModules...)
	var extras []string
	for _, mod := range allowedModules {
		if slices.Contains(AllExtraModuleNames(), mod) {
			extras = append(extras, mod)
		}
	}
	extraMap := GetExtraModuleMap(logger, extras...)
	moduleMap.AddMap(extraMap)
	return moduleMap
}
