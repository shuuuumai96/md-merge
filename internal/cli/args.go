package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shuuuumai96/md-merge/internal/merge"
	"github.com/shuuuumai96/md-merge/internal/scan"
)

func ParseArgs(args []string) (merge.Config, int, error) {
	cfg := merge.Config{
		Extensions: scan.ParseExtensions("md,markdown"),
		SortMode:   merge.SortByPath,
	}
	var rootSeen bool

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-o", "--output":
			if i+1 >= len(args) {
				return cfg, ExitInvalidInput, errors.New("missing value for output flag")
			}
			i++
			cfg.OutputPath = args[i]
		case "--dry-run":
			cfg.DryRun = true
		case "--with-toc":
			cfg.WithTOC = true
		case "--exclude":
			if i+1 >= len(args) {
				return cfg, ExitInvalidInput, errors.New("missing value for exclude flag")
			}
			i++
			cfg.Excludes = append(cfg.Excludes, args[i])
		case "--extensions":
			if i+1 >= len(args) {
				return cfg, ExitInvalidInput, errors.New("missing value for extensions flag")
			}
			i++
			cfg.Extensions = scan.ParseExtensions(args[i])
			if len(cfg.Extensions) == 0 {
				return cfg, ExitInvalidInput, errors.New("extensions list cannot be empty")
			}
		case "--sort":
			if i+1 >= len(args) {
				return cfg, ExitInvalidInput, errors.New("missing value for sort flag")
			}
			i++
			cfg.SortMode = merge.SortMode(args[i])
		case "--strict":
			cfg.Strict = true
		case "--max-size":
			if i+1 >= len(args) {
				return cfg, ExitInvalidInput, errors.New("missing value for max-size flag")
			}
			i++
			value, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil || value < 0 {
				return cfg, ExitInvalidInput, errors.New("max-size must be a non-negative integer")
			}
			cfg.MaxSizeBytes = value
		default:
			if strings.HasPrefix(arg, "-") {
				return cfg, ExitInvalidInput, fmt.Errorf("unknown flag %s", arg)
			}
			if rootSeen {
				return cfg, ExitInvalidInput, errors.New("only one target directory is supported")
			}
			cfg.RootDir = arg
			rootSeen = true
		}
	}
	if !rootSeen {
		return cfg, ExitInvalidInput, errors.New("target directory is required")
	}
	return cfg, ExitSuccess, nil
}

func NormalizeConfig(cfg merge.Config) (merge.Config, error) {
	if cfg.RootDir == "" {
		return cfg, errors.New("target directory is required")
	}

	rootAbs, err := filepath.Abs(filepath.Clean(cfg.RootDir))
	if err != nil {
		return cfg, fmt.Errorf("resolve target directory: %w", err)
	}
	info, err := os.Stat(rootAbs)
	if err != nil {
		return cfg, fmt.Errorf("target directory is invalid: %w", err)
	}
	if !info.IsDir() {
		return cfg, errors.New("target path is not a directory")
	}
	cfg.RootDir = rootAbs

	if cfg.OutputPath != "" {
		outputAbs, err := filepath.Abs(filepath.Clean(cfg.OutputPath))
		if err != nil {
			return cfg, fmt.Errorf("resolve output path: %w", err)
		}
		cfg.OutputPath = outputAbs
	}

	if len(cfg.Extensions) == 0 {
		cfg.Extensions = scan.ParseExtensions("md,markdown")
	}
	if cfg.SortMode == "" {
		cfg.SortMode = merge.SortByPath
	}
	if !merge.ValidSortMode(cfg.SortMode) {
		return cfg, fmt.Errorf("invalid sort mode %q", cfg.SortMode)
	}

	merged := make([]string, 0, len(merge.DefaultExcludes)+len(cfg.Excludes))
	merged = append(merged, merge.DefaultExcludes...)
	merged = append(merged, cfg.Excludes...)
	cfg.Excludes = merged

	return cfg, nil
}
