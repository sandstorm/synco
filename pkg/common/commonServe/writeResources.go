package commonServe

import (
	"encoding/json"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
)

func WriteResourcesIndex(transferSession *serve.TransferSession, name string, resourceFilesIndex dto.PublicFilesIndex, totalSizeBytes uint64) {
	indexFileName := "Resources.index.json.enc"

	bytes, err := json.Marshal(resourceFilesIndex)
	if err != nil {
		pterm.Fatal.Printfln("could not encode resourceFilesIndex: %s", err)
	}

	err = transferSession.EncryptBytesToFile(indexFileName, bytes)
	if err != nil {
		pterm.Fatal.Printfln("could not encrypt to file: %s", err)
	}

	fileSet := &dto.FileSet{
		Name: name,
		Type: dto.TYPE_PUBLICFILES,
		PublicFiles: &dto.FileSetPublicFiles{
			IndexFileName: indexFileName,
			SizeBytes:     totalSizeBytes,
		},
	}
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update Resource dump metadata: %s", err)
	}
	pterm.Info.Printfln("Extracted Resource Index")
}
