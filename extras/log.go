package extras

import (
	"errors"
	"log/slog"

	"github.com/d5/tengo/v2"
)

// logModule creates a custom Tengo log module.
func logModule(logger *slog.Logger) map[string]tengo.Object {
	return map[string]tengo.Object{
		"debug": &tengo.UserFunction{
			Name: "debug",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return nil, errors.New("log.debug: expected at least one argument")
				}
				msg, ok := args[0].(*tengo.String)
				if !ok {
					return nil, errors.New("log.debug: first argument must be a string")
				}
				if len(args[1:])%2 != 0 {
					return nil, errors.New("log.debug: key-value pairs must be even in number")
				}
				var kv []any
				for i := 1; i < len(args); i += 2 {
					k, ok := args[i].(*tengo.String)
					v, _ := args[i+1].(*tengo.String)
					if !ok {
						return nil, errors.New("log.debug: key must be a string")
					}
					kv = append(kv, k.Value, v.Value)
				}
				logger.Debug(msg.Value, kv...)
				return nil, nil
			},
		},
		"info": &tengo.UserFunction{
			Name: "info",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return nil, errors.New("log.info: expected at least one argument")
				}
				msg, ok := args[0].(*tengo.String)
				if !ok {
					return nil, errors.New("log.info: first argument must be a string")
				}
				if len(args[1:])%2 != 0 {
					return nil, errors.New("log.info: key-value pairs must be even in number")
				}
				var kv []any
				for i := 1; i < len(args); i += 2 {
					k, ok := args[i].(*tengo.String)
					v, _ := args[i+1].(*tengo.String)
					if !ok {
						return nil, errors.New("log.info: key must be a string")
					}
					kv = append(kv, k.Value, v.Value)
				}
				logger.Info(msg.Value, kv...)
				return nil, nil
			},
		},
		"warn": &tengo.UserFunction{
			Name: "warn",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return nil, errors.New("log.warn: expected at least one argument")
				}
				msg, ok := args[0].(*tengo.String)
				if !ok {
					return nil, errors.New("log.warn: first argument must be a string")
				}
				if len(args[1:])%2 != 0 {
					return nil, errors.New("log.warn: key-value pairs must be even in number")
				}
				var kv []any
				for i := 1; i < len(args); i += 2 {
					k, ok := args[i].(*tengo.String)
					v, _ := args[i+1].(*tengo.String)
					if !ok {
						return nil, errors.New("log.warn: key must be a string")
					}
					kv = append(kv, k.Value, v.Value)
				}
				logger.Warn(msg.Value, kv...)
				return nil, nil
			},
		},
		"error": &tengo.UserFunction{
			Name: "error",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return nil, errors.New("log.error: expected at least one argument")
				}
				msg, ok := args[0].(*tengo.String)
				if !ok {
					return nil, errors.New("log.error: first argument must be a string")
				}
				if len(args[1:])%2 != 0 {
					return nil, errors.New("log.error: key-value pairs must be even in number")
				}
				var kv []any
				for i := 1; i < len(args); i += 2 {
					k, ok := args[i].(*tengo.String)
					v, _ := args[i+1].(*tengo.String)
					if !ok {
						return nil, errors.New("log.error: key must be a string")
					}
					kv = append(kv, k.Value, v.Value)
				}
				logger.Error(msg.Value, kv...)
				return nil, nil
			},
		},
	}
}
