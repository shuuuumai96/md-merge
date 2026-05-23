package merge

import "time"

type FileRecord struct {
	AbsPath     string
	RelPath     string
	DisplayPath string
	Name        string
	SizeBytes   int64
	ModifiedAt  time.Time
}

type ScanResult struct {
	Files    []FileRecord
	Warnings []Warning
}
