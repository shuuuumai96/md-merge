package scan

import "strings"

func ParseExtensions(list string) map[string]bool {
	extensions := map[string]bool{}
	for _, item := range strings.Split(list, ",") {
		ext := strings.TrimPrefix(strings.TrimSpace(strings.ToLower(item)), ".")
		if ext != "" {
			extensions[ext] = true
		}
	}
	return extensions
}
