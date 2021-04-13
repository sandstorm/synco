package marshal

import (
	"time"
)

const (
	MarkerTable byte = 231 + iota
	MarkerRow
)

type FileHeader struct {
	ServerVersion string
	DatabaseName  string
	DumpStart     time.Time
}

type TableHeader struct {
	Name      string
	CreateSQL string
}

type RowData = []*string
