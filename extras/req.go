package extras

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/d5/tengo/v2"
	req "github.com/imroc/req/v3"
)

func reqModule(logger *slog.Logger) map[string]tengo.Object {
	client := req.C().ImpersonateChrome()
	return map[string]tengo.Object{
		"get": &tengo.UserFunction{
			Name: "get",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("http.get: expected 1 argument")
				}
				urlStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("http.get: argument must be a string")
				}
				headers := map[string]string{}
				if len(args) == 2 {
					hdrMap, ok := args[1].(*tengo.Map)
					if !ok {
						return nil, fmt.Errorf("http.get: second argument must be a map")
					}
					for k, v := range hdrMap.Value {
						headers[k] = strings.Trim(v.String(), `"`)
					}
				}
				var r *req.Response
				var err error
				for i := range 2 {
					r, err = client.R().SetHeaders(headers).Get(urlStr.Value)
					if err != nil {
						logger.Warn("http.get: retry", "attempt", i+1, "error", err)
						continue
					}
					break
				}
				if err != nil {
					return nil, fmt.Errorf("http.get: %w", err)
				}
				result := map[string]tengo.Object{
					"status":  &tengo.Int{Value: int64(r.Response.StatusCode)},
					"body":    &tengo.String{Value: r.String()},
					"headers": convertHeaders(r.Response.Header),
				}
				return &tengo.Map{Value: result}, nil
			},
		},
		"post": &tengo.UserFunction{
			Name: "post",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 || len(args) > 3 {
					return nil, fmt.Errorf("http.post: expected 2 or 3 arguments")
				}
				urlStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("http.post: first argument must be a string")
				}
				dataStr, ok := args[1].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("http.post: second argument must be a string")
				}
				headers := map[string]string{}
				if len(args) == 3 {
					hdrMap, ok := args[2].(*tengo.Map)
					if !ok {
						return nil, fmt.Errorf("http.post: third argument must be a map")
					}
					for k, v := range hdrMap.Value {
						headers[k] = strings.Trim(v.String(), `"`)
					}
				}
				var r *req.Response
				var err error
				for i := range 2 {
					r, err = client.R().SetHeaders(headers).SetBody(dataStr.Value).Post(urlStr.Value)
					if err != nil {
						logger.Warn("http.post: retry", "attempt", i+1, "error", err)
						continue
					}
					break
				}
				if err != nil {
					return nil, fmt.Errorf("http.post: %w", err)
				}
				result := map[string]tengo.Object{
					"status":  &tengo.Int{Value: int64(r.Response.StatusCode)},
					"body":    &tengo.String{Value: r.String()},
					"headers": convertHeaders(r.Response.Header),
				}
				return &tengo.Map{Value: result}, nil
			},
		},
	}
}

// convertHeaders converts http.Header to a Tengo map.
func convertHeaders(hdr map[string][]string) *tengo.Map {
	m := make(map[string]tengo.Object, len(hdr))
	for k, v := range hdr {
		m[k] = &tengo.Array{Value: stringsToTengoArray(v)}
	}
	return &tengo.Map{Value: m}
}

func stringsToTengoArray(strs []string) []tengo.Object {
	arr := make([]tengo.Object, len(strs))
	for i, s := range strs {
		arr[i] = &tengo.String{Value: s}
	}
	return arr
}
