// Package extras provides extra Tengo modules.
package extras

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/d5/tengo/v2"
)

// miscModule implements the novel module.
func miscModule(logger *slog.Logger) map[string]tengo.Object {
	return map[string]tengo.Object{
		"title_clean": &tengo.UserFunction{
			Name: "title_clean",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.title_clean: expected 1 argument")
				}
				title, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("novel.title_clean: argument must be a string")
				}
				clean := strings.TrimSpace(title.Value)
				suffixes := []string{" - Novel", " - Volume", " Novel"}
				for _, sfx := range suffixes {
					clean = strings.TrimSuffix(clean, sfx)
				}
				return &tengo.String{Value: clean}, nil
			},
		},
		"slugify": &tengo.UserFunction{
			Name: "slugify",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.slugify: expected 1 argument")
				}
				s, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("novel.slugify: argument must be a string")
				}
				slug := strings.ToLower(strings.ReplaceAll(s.Value, " ", "-"))
				return &tengo.String{Value: slug}, nil
			},
		},
		"chapter_number": &tengo.UserFunction{
			Name: "chapter_number",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.chapter_number: expected 1 argument")
				}
				title, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("novel.chapter_number: argument must be a string")
				}
				re := regexp.MustCompile(`\d+`)
				match := re.FindString(title.Value)
				if match == "" {
					return &tengo.Int{Value: 0}, nil
				}
				var num int
				fmt.Sscanf(match, "%d", &num)
				return &tengo.Int{Value: int64(num)}, nil
			},
		},
		"absolute_url": &tengo.UserFunction{
			Name: "absolute_url",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("novel.absolute_url: expected 2 arguments")
				}
				baseStr, ok1 := args[0].(*tengo.String)
				relStr, ok2 := args[1].(*tengo.String)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("novel.absolute_url: both arguments must be strings")
				}
				baseURL, err := url.Parse(baseStr.Value)
				if err != nil {
					return nil, fmt.Errorf("novel.absolute_url: %w", err)
				}
				relURL, err := url.Parse(relStr.Value)
				if err != nil {
					return nil, fmt.Errorf("novel.absolute_url: %w", err)
				}
				absURL := baseURL.ResolveReference(relURL)
				return &tengo.String{Value: absURL.String()}, nil
			},
		},
		"is_chapter_url": &tengo.UserFunction{
			Name: "is_chapter_url",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.is_chapter_url: expected 1 argument")
				}
				urlStr, ok := args[0].(*tengo.String)
				if !ok {
					return nil, fmt.Errorf("novel.is_chapter_url: argument must be a string")
				}
				isChapter := strings.Contains(strings.ToLower(urlStr.Value), "chapter")
				if isChapter {
					return tengo.TrueValue, nil
				}
				return tengo.FalseValue, nil
			},
		},
		"filter_chapter_links": &tengo.UserFunction{
			Name: "filter_chapter_links",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.filter_chapter_links: expected 1 argument")
				}
				arr, ok := args[0].(*tengo.Array)
				if !ok {
					return nil, fmt.Errorf("novel.filter_chapter_links: argument must be an array")
				}
				var filtered []tengo.Object
				for _, item := range arr.Value {
					str, ok := item.(*tengo.String)
					if !ok {
						continue
					}
					if strings.Contains(strings.ToLower(str.Value), "chapter") {
						filtered = append(filtered, str)
					}
				}
				return &tengo.Array{Value: filtered}, nil
			},
		},
		"sort_chapters": &tengo.UserFunction{
			Name: "sort_chapters",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("novel.sort_chapters: expected 1 argument")
				}
				arr, ok := args[0].(*tengo.Array)
				if !ok {
					return nil, fmt.Errorf("novel.sort_chapters: argument must be an array")
				}
				var chapters []map[string]any
				for _, item := range arr.Value {
					m, ok := item.(*tengo.Map)
					if !ok {
						continue
					}
					goMap := make(map[string]any)
					for k, v := range m.Value {
						goMap[fmt.Sprintf("%v", k)] = v.String()
					}
					chapters = append(chapters, goMap)
				}
				sort.Slice(chapters, func(i, j int) bool {
					re := regexp.MustCompile(`\d+`)
					mi := re.FindString(fmt.Sprintf("%v", chapters[i]["title"]))
					mj := re.FindString(fmt.Sprintf("%v", chapters[j]["title"]))
					var ni, nj int
					fmt.Sscanf(mi, "%d", &ni)
					fmt.Sscanf(mj, "%d", &nj)
					return ni < nj
				})
				var result []tengo.Object
				for _, ch := range chapters {
					tempMap := make(map[string]tengo.Object)
					for k, v := range ch {
						tempMap[k] = &tengo.String{Value: fmt.Sprintf("%v", v)}
					}
					result = append(result, &tengo.Map{Value: tempMap})
				}
				return &tengo.Array{Value: result}, nil
			},
		},
	}
}
