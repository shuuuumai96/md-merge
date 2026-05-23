package output

import (
	"io"
	"os"
	"path/filepath"

	"github.com/shuuuumai96/md-merge/internal/merge"
)

func Write(cfg merge.Config, stdout io.Writer, data []byte) error {
	if cfg.OutputPath == "" {
		_, err := stdout.Write(data)
		return err
	}

	dir := filepath.Dir(cfg.OutputPath)
	tmp, err := os.CreateTemp(dir, ".md-merge-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, cfg.OutputPath)
}
