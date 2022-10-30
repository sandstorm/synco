package dto

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"strings"
)

const (
	FILENAME_META = "meta.json.enc"
)

type State string

const (
	STATE_CREATED      State = "Created"
	STATE_INITIALIZING State = "Initializing"
	STATE_READY        State = "Ready"
)

// Meta is the main structure which is serialized into meta.json.enc files. It contains
// the entry points for an exported file.
type Meta struct {
	State         State      `json:"state"`
	FrameworkName string     `json:"frameworkName"`
	FileSets      []*FileSet `json:"fileSets"`
}

func (m Meta) FileSetByLabel(label string) *FileSet {
	name := extractNameFromLabel(label)
	for _, fileSet := range m.FileSets {
		if fileSet.Name == name {
			return fileSet
		}
	}
	return nil
}

type FileSetType string

const (
	TYPE_MYSQLDUMP    FileSetType = "MysqlDump"
	TYPE_POSTGRESDUMP FileSetType = "PostgresDump"
	TYPE_PUBLICFILES  FileSetType = "PublicFiles"
)

type FileSet struct {
	Name         string               `json:"name"`
	Type         FileSetType          `json:"type"`
	MysqlDump    *FileSetMysqlDump    `json:"mysqlDump"`
	PostgresDump *FileSetPostgresDump `json:"postgresDump"`
	PublicFiles  *FileSetPublicFiles  `json:"publicFiles"`
}

func (fileSet *FileSet) Label() string {

	switch fileSet.Type {
	case TYPE_MYSQLDUMP:
		return fmt.Sprintf("%s (%s: %s)", fileSet.Name, fileSet.Type, humanize.IBytes(fileSet.MysqlDump.SizeBytes))
	case TYPE_POSTGRESDUMP:
		return fmt.Sprintf("%s (%s: %s)", fileSet.Name, fileSet.Type, humanize.IBytes(fileSet.PostgresDump.SizeBytes))
	case TYPE_PUBLICFILES:
		return fmt.Sprintf("%s (%s: %s)", fileSet.Name, fileSet.Type, humanize.IBytes(fileSet.PublicFiles.SizeBytes))
	default:
		return fmt.Sprintf("%s (%s)", fileSet.Name, fileSet.Type)
	}
}

func extractNameFromLabel(label string) string {
	tmp := strings.SplitN(label, " (", 2)
	return tmp[0]
}

type FileSetMysqlDump struct {
	FileName  string `json:"fileName"`
	SizeBytes uint64 `json:"sizeBytes"`
}

type FileSetPostgresDump struct {
	FileName  string `json:"fileName"`
	SizeBytes uint64 `json:"sizeBytes"`
}

type FileSetPublicFiles struct {
	IndexFileName string `json:"indexFileName"`
	SizeBytes     uint64 `json:"sizeBytes"`
}

// PublicFilesIndex is the structure of the "index" file for FileSetPublicFiles.
type PublicFilesIndex map[string]PublicFilesIndexEntry

type PublicFilesIndexEntry struct {
}
