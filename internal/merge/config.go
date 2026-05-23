package merge

type Config struct {
	RootDir      string
	OutputPath   string
	Extensions   map[string]bool
	Excludes     []string
	SortMode     SortMode
	WithTOC      bool
	DryRun       bool
	Strict       bool
	MaxSizeBytes int64
}

var DefaultExcludes = []string{
	".git",
	"node_modules",
	"dist",
	"build",
	"coverage",
	"vendor",
	".cache",
}
