package render

import (
	"bytes"
	"fmt"

	"github.com/shuuuumai96/md-merge/internal/merge"
)

func DryRun(files []merge.FileRecord) []byte {
	var out bytes.Buffer
	for _, file := range files {
		fmt.Fprintln(&out, file.DisplayPath)
	}
	return out.Bytes()
}
