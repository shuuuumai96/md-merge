package scan

import (
	"path/filepath"
	"strings"
)

func ShouldExclude(displayPathValue, name string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(filepath.ToSlash(pattern))
		if pattern == "" {
			continue
		}
		if !strings.ContainsAny(pattern, "*/?[") && pathHasSegment(displayPathValue, pattern) {
			return true
		}
		if globMatch(pattern, displayPathValue) || globMatch(strings.Trim(pattern, "/"), name) {
			return true
		}
	}
	return false
}

func pathHasSegment(displayPathValue, segment string) bool {
	for _, part := range strings.Split(displayPathValue, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

func globMatch(pattern, value string) bool {
	if pattern == value {
		return true
	}
	if strings.HasPrefix(pattern, "**/") {
		rest := strings.TrimPrefix(pattern, "**/")
		if ok, _ := filepath.Match(rest, value); ok {
			return true
		}
		for i := range value {
			if value[i] == '/' {
				if globMatch(rest, value[i+1:]) {
					return true
				}
			}
		}
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return value == prefix || strings.HasPrefix(value, prefix+"/")
	}
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			return strings.HasPrefix(value, strings.TrimSuffix(parts[0], "/")) &&
				strings.HasSuffix(value, strings.TrimPrefix(parts[1], "/"))
		}
	}
	ok, _ := filepath.Match(pattern, value)
	return ok
}
