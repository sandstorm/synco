package commonServe

import (
	"encoding/json"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/serve"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ExtractAllResourcesFromFolder(transferSession *serve.TransferSession, persistentResourcesBasePath string, baseUri string) {
	resourceFilesIndex := make(dto.PublicFilesIndex)
	totalSizeBytes := uint64(0)
	err := filepath.Walk(persistentResourcesBasePath,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// skip directories on traversal
				return nil
			}

			realPath, err := filepath.EvalSymlinks(filePath)
			if err != nil {
				pterm.Error.Printfln("Could NOT evaluate symlinks (skipping): %s: %s", filePath, err)
				return nil
			}
			realFileInfo, err := os.Lstat(realPath)
			if err != nil {
				pterm.Error.Printfln("Could NOT read file info (skipping): %s: %s", realPath, err)
				return nil
			}

			filePath = strings.TrimPrefix(filePath, persistentResourcesBasePath)

			// Flow stores files in /..../<resourceSha1>/<filename>.jpg; so we extract the resourceSha1 here.
			resourceSha1 := path.Base(path.Dir(filePath))

			publicUri, err := url.JoinPath(baseUri, filePath)
			if err != nil {
				return err
			}

			totalSizeBytes += uint64(realFileInfo.Size())
			resourceFilesIndex["Resources/"+resourceSha1[0:1]+"/"+resourceSha1[1:2]+"/"+resourceSha1[2:3]+"/"+resourceSha1[3:4]+"/"+resourceSha1] = dto.PublicFilesIndexEntry{
				SizeBytes: int64(realFileInfo.Size()),
				MTime:     realFileInfo.ModTime().Unix(),
				PublicUri: "<BASE>/" + publicUri,
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	WriteResourcesIndex(transferSession, resourceFilesIndex, totalSizeBytes)
}

func WriteResourcesIndex(transferSession *serve.TransferSession, resourceFilesIndex dto.PublicFilesIndex, totalSizeBytes uint64) {
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
		Name: "Resources",
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
