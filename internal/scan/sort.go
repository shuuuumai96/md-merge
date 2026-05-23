package scan

import (
	"sort"

	"github.com/shuuuumai96/md-merge/internal/merge"
)

type SortFunc func([]merge.FileRecord)

var sortFuncs = map[merge.SortMode]SortFunc{
	merge.SortByPath:     SortByPath,
	merge.SortByName:     SortByName,
	merge.SortByModified: SortByModified,
}

func SortFiles(files []merge.FileRecord, mode merge.SortMode) {
	sortFunc := sortFuncs[mode]
	if sortFunc == nil {
		sortFunc = SortByPath
	}
	sortFunc(files)
}

func SortByPath(files []merge.FileRecord) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].DisplayPath < files[j].DisplayPath
	})
}

func SortByName(files []merge.FileRecord) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Name != files[j].Name {
			return files[i].Name < files[j].Name
		}
		return files[i].DisplayPath < files[j].DisplayPath
	})
}

func SortByModified(files []merge.FileRecord) {
	sort.Slice(files, func(i, j int) bool {
		if !files[i].ModifiedAt.Equal(files[j].ModifiedAt) {
			return files[i].ModifiedAt.Before(files[j].ModifiedAt)
		}
		return files[i].DisplayPath < files[j].DisplayPath
	})
}
