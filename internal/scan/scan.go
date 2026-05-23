package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuuuumai96/md-merge/internal/merge"
)

func MarkdownFiles(cfg merge.Config) (merge.ScanResult, error) {
	var result merge.ScanResult
	outputAbs := filepath.Clean(cfg.OutputPath)

	err := filepath.WalkDir(cfg.RootDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warning := merge.Warning{Path: DisplayPath(cfg.RootDir, path), Message: walkErr.Error()}
			if cfg.Strict {
				return fmt.Errorf("%s: %s", warning.Path, warning.Message)
			}
			result.Warnings = append(result.Warnings, warning)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if path == cfg.RootDir {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			warning := merge.Warning{Path: DisplayPath(cfg.RootDir, path), Message: err.Error()}
			if cfg.Strict {
				return fmt.Errorf("%s: %s", warning.Path, warning.Message)
			}
			result.Warnings = append(result.Warnings, warning)
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(cfg.RootDir, path)
		if err != nil {
			return err
		}
		display := filepath.ToSlash(rel)

		if entry.IsDir() {
			if ShouldExclude(display, entry.Name(), cfg.Excludes) {
				return filepath.SkipDir
			}
			return nil
		}
		if ShouldExclude(display, entry.Name(), cfg.Excludes) {
			return nil
		}
		if cfg.OutputPath != "" && filepath.Clean(path) == outputAbs {
			return nil
		}
		if !cfg.Extensions[strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")] {
			return nil
		}

		result.Files = append(result.Files, merge.FileRecord{
			AbsPath:     filepath.Clean(path),
			RelPath:     rel,
			DisplayPath: display,
			Name:        entry.Name(),
			SizeBytes:   info.Size(),
			ModifiedAt:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return result, err
	}

	SortFiles(result.Files, cfg.SortMode)
	return result, nil
}

func DisplayPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
