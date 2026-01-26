package render

import (
	"html/template"
	"strings"
)

// FuncMap returns a template.FuncMap with all render functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// String
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,

		// Math
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"seq": func(start, end int) []int {
			if end < start {
				return nil
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},

		// HTML
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeAttr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},

		// String manipulation
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace":   strings.Replace,
		"split":     strings.Split,
		"join":      strings.Join,

		// Comparisons
		"eq": func(a, b any) bool { return a == b },
		"ne": func(a, b any) bool { return a != b },
		"lt": func(a, b int) bool { return a < b },
		"le": func(a, b int) bool { return a <= b },
		"gt": func(a, b int) bool { return a > b },
		"ge": func(a, b int) bool { return a >= b },

		// Collections
		"first": func(items []any) any {
			if len(items) > 0 {
				return items[0]
			}
			return nil
		},
		"last": func(items []any) any {
			if len(items) > 0 {
				return items[len(items)-1]
			}
			return nil
		},
		"len": func(items any) int {
			switch v := items.(type) {
			case []any:
				return len(v)
			case string:
				return len(v)
			default:
				return 0
			}
		},
	}
}

// MergeFuncMaps merges multiple FuncMaps into one.
// Later maps override earlier ones for duplicate keys.
func MergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	result := make(template.FuncMap)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
