package merge

type SortMode string

const (
	SortByPath     SortMode = "path"
	SortByName     SortMode = "name"
	SortByModified SortMode = "modified"
)

func ValidSortMode(mode SortMode) bool {
	switch mode {
	case SortByPath, SortByName, SortByModified:
		return true
	default:
		return false
	}
}
