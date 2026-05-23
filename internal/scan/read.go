package scan

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/shuuuumai96/md-merge/internal/merge"
)

func ReadFileContents(files []merge.FileRecord, cfg merge.Config) (map[string]string, []merge.Warning, error) {
	contents := make(map[string]string, len(files))
	var warnings []merge.Warning

	for _, file := range files {
		if cfg.MaxSizeBytes > 0 && file.SizeBytes > cfg.MaxSizeBytes {
			warning := merge.Warning{Path: file.DisplayPath, Message: fmt.Sprintf("file exceeds max size %d bytes", cfg.MaxSizeBytes)}
			if cfg.Strict {
				return nil, warnings, fmt.Errorf("%s: %s", warning.Path, warning.Message)
			}
			warnings = append(warnings, warning)
			continue
		}

		data, err := os.ReadFile(file.AbsPath)
		if err != nil {
			warning := merge.Warning{Path: file.DisplayPath, Message: err.Error()}
			if cfg.Strict {
				return nil, warnings, fmt.Errorf("%s: %s", warning.Path, warning.Message)
			}
			warnings = append(warnings, warning)
			continue
		}
		if !utf8.Valid(data) {
			warning := merge.Warning{Path: file.DisplayPath, Message: "invalid UTF-8"}
			if cfg.Strict {
				return nil, warnings, fmt.Errorf("%s: %s", warning.Path, warning.Message)
			}
			warnings = append(warnings, warning)
			continue
		}
		contents[file.DisplayPath] = string(data)
	}

	return contents, warnings, nil
}
