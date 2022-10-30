package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/repeale/fp-go"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/receive"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var interactive bool

var ReceiveCmd = &cobra.Command{
	Use:     "receive",
	Short:   "Wizard to be executed in target",
	Long:    `...`,
	Args:    cobra.ExactArgs(2),
	Example: `synco receive [url] [password]`,
	// Uncomment the following lines if your bare application has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		baseUrl := args[0]
		password := args[1]

		receiveSession, err := receive.NewSession(baseUrl, password)
		pterm.PrintOnErrorf("Error initializing receive session: %e", err)
		if err != nil {
			return
		}

		meta, err := receiveSession.FetchMeta()
		if err != nil {
			pterm.Fatal.Printfln("Metadata could not be fetched: %s", err)
		}
		pterm.Success.Printfln("Valid Decryption Key")
		pterm.Success.Printfln("Framework on server: %s", meta.FrameworkName)

		if meta.State != dto.STATE_READY {
			pterm.Error.Printfln("Received state '%s', but you need to run the tool once Ready state exists. Please re-run!", meta.State)
			os.Exit(1)
		}

		filesToDownload := fp.Map(func(fileSet *dto.FileSet) string {
			return fileSet.Label()
		})(meta.FileSets)

		if interactive {
			filesToDownload, err = pterm.DefaultInteractiveMultiselect.
				WithOptions(filesToDownload).
				WithDefaultOptions(filesToDownload).
				Show()
			if err != nil {
				pterm.Fatal.Printfln("File Selector could not be shown: %s", err)
			}
		}

		for _, fileToDownload := range filesToDownload {
			fileSet := meta.FileSetByLabel(fileToDownload)
			pterm.Info.Printfln("Downloading: %s", fileToDownload, fileSet.Type)

			switch fileSet.Type {
			case dto.TYPE_MYSQLDUMP:
				err = downloadMysqldump(receiveSession, fileSet)
			case dto.TYPE_PUBLICFILES:
				err = downloadPublicFiles(receiveSession, fileSet)
			default:
				pterm.Fatal.Printfln("File Set type %s was unimplemented.", fileSet.Type)
			}

			if err != nil {
				pterm.Fatal.Printfln("Error with file type %s: %s", fileSet.Type, err)
			}
		}

		/*for _, framework := range RegisteredFrameworks {
			if framework.Name() == meta.frameworkName {
				framework.Receive(receiveSession)
				return
			}
		}
		pterm.Error.Printfln("Framework %s implementation not detected on client side. This means your synchro client needs to be upgraded to match the server side version", pterm.ThemeDefault.HighlightStyle.Sprint(frameworkName))
		os.Exit(1)*/
	},
}

func downloadMysqldump(receiveSession *receive.ReceiveSession, fileSet *dto.FileSet) error {
	return receiveSession.DumpAndDecryptFileWithProgressBar(fileSet.MysqlDump.FileName, fileSet.Name+".sql")
}

func downloadPublicFiles(receiveSession *receive.ReceiveSession, fileSet *dto.FileSet) error {
	indexFileName := fileSet.Name + ".index.json"
	err := receiveSession.DumpAndDecryptFileWithProgressBar(fileSet.PublicFiles.IndexFileName, indexFileName)
	if err != nil {
		return fmt.Errorf("error dumping/decrypting files: %w", err)
	}

	indexBytes, err := receiveSession.FileContentsInWorkDir(indexFileName)
	if err != nil {
		return fmt.Errorf("error reading file contents in workdir: %w", err)
	}

	var publicFilesIndex dto.PublicFilesIndex
	err = json.Unmarshal(indexBytes, &publicFilesIndex)
	if err != nil {
		return fmt.Errorf("error unmarshalling %s: %w", indexFileName, err)
	}

	for fileName, fileDefinition := range publicFilesIndex {
		// create parent directory
		err = os.MkdirAll(filepath.Dir(fileName), 0755)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", filepath.Dir(fileName), err)
		}

		// Check for changes of the files (based on size and modification times)
		fileStat, err := receiveSession.StatInWorkDir(fileName)
		if err == nil {
			if fileStat.Size() == fileDefinition.SizeBytes && fileStat.ModTime().Unix() == fileDefinition.MTime {
				// file exists; and exists with same size and modification time. We can skip the download.
				pterm.Debug.Printfln("Ignoring file %s, because it exists already with same size and modification timestamp", fileName)
				continue
			} else if fileStat.Size() != fileDefinition.SizeBytes {
				pterm.Debug.Printfln("Re-downloading file %s, because file sizes do not match: %d (local) != %d (remote)", fileName, fileStat.Size(), fileDefinition.SizeBytes)
			} else if fileStat.ModTime().Unix() != fileDefinition.MTime {
				pterm.Debug.Printfln("Re-downloading file %s, because file modification times do not match: %d (local) != %d (remote)", fileName, fileStat.ModTime().Unix(), fileDefinition.MTime)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			// some other error except "file does not exist" => bubble up
			return fmt.Errorf("error calling stat on %s: %w", fileName, err)
		}

		// download file.
		err = receiveSession.DumpFileWithProgressBar(strings.ReplaceAll(fileDefinition.PublicUri, "<BASE>", ".."), fileName)
		if err != nil {
			return fmt.Errorf("error on downloading %s to %s: %w", fileDefinition.PublicUri, fileName, err)
		}
		// set the desired modification time to the server's modification time (for change tracking)
		desiredMtime := time.Unix(fileDefinition.MTime, 0)
		pterm.Debug.Printfln("Setting mtime for %s to %d", fileName, fileDefinition.MTime)
		err = receiveSession.SetMTimeInWorkDir(fileName, desiredMtime)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	ReceiveCmd.Flags().BoolVar(&interactive, "interactive", true, "identifier for the decryption")
}
