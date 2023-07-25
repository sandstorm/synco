package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/repeale/fp-go"
	"github.com/sandstorm/synco/pkg/common/config"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/receive"
	"github.com/sandstorm/synco/pkg/ui/boolselect"
	"github.com/sandstorm/synco/pkg/ui/multiselect"
	"github.com/sandstorm/synco/pkg/ui/textinput"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"strings"
	"time"
)

var interactive bool

var ReceiveCmd = &cobra.Command{
	Use:     "receive",
	Short:   "Wizard to be executed in target",
	Long:    `...`,
	Args:    cobra.ExactArgs(2),
	Example: `synco receive [identifier] [password]`,
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]
		password := args[1]

		receiveSession, err := receive.NewSession(identifier, password)
		pterm.PrintOnErrorf("Error initializing receive session: %s", err)
		if err != nil {
			return
		}

		err = detectBaseUrlAndUpdateReceiveSession(receiveSession)
		if err != nil {
			pterm.Error.Printfln("Error detecting base URL: %s", err)
			return
		}

		meta, err := receiveSession.FetchMeta()
		if err != nil {
			pterm.Error.Printfln("Metadata could not be fetched: %s", err)
			return
		}
		pterm.Success.Printfln("Valid Decryption Key")
		pterm.Info.Printfln("Framework on server: %s", meta.FrameworkName)

		if meta.State != dto.STATE_READY {
			pterm.Error.Printfln("Received state '%s', but you need to run the tool once Ready state exists. Please re-run!", meta.State)
			os.Exit(1)
		}

		filesToDownload := fp.Map(func(fileSet *dto.FileSet) string {
			return fileSet.Label()
		})(meta.FileSets)

		if interactive {
			filesToDownload = multiselect.Exec("Select data to download", filesToDownload, filesToDownload)
		}

		for _, fileToDownload := range filesToDownload {
			fileSet := meta.FileSetByLabel(fileToDownload)
			pterm.Info.Printfln("Downloading: %s (%s)", fileToDownload, fileSet.Type)

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

		pterm.Success.Printfln("All downloaded to dump/")

		/*for _, framework := range RegisteredFrameworks {
			if framework.Name() == meta.frameworkName {
				framework.Receive(receiveSession)
				return
			}
		}
		pterm.Error.Printfln("Framework %s implementation not detected on client side. This means your synchro client needs to be upgraded to match the server side version", pterm.ThemeDefault.HighlightStyle.Sprint(frameworkName))
		*/

		pterm.Success.Printfln("FINISHED :) Now, terminate %s on the server side by pressing %s.", pterm.ThemeDefault.PrimaryStyle.Sprint("synco serve"), pterm.ThemeDefault.PrimaryStyle.Sprint("Ctrl-C"))
		os.Exit(0)
	},
}

// detectBaseUrlAndUpdateReceiveSession tries to find the base URL, by:
// - reading .synco.yml
// - asking the user
// - validating whether the host can be found.
func detectBaseUrlAndUpdateReceiveSession(rs *receive.ReceiveSession) error {
	syncoConfig, err := config.ReadFromYaml()
	if err != nil {
		return err
	}
	for _, host := range syncoConfig.Hosts {
		pterm.Debug.Printfln("Trying to detect the base URL: %s", host.BaseUrl)
		rs.BaseUrl(host.BaseUrl)
		err = rs.DoesMetaFileExistOnServer()
		if err == nil {
			// we found the meta file; so we are done.
			pterm.Success.Printfln("Found correct base URL at %s (based on %s)", host.BaseUrl, config.SyncoYamlFile)
			// NOTE: the receiveSession is already updated; so we do not need to update anything.
			return nil
		}
		if !errors.Is(err, receive.ErrMetaFileNotFound) {
			// we got an unexpected error -> bubble up
			return err
		}
		// receive.ErrMetaFileNotFound -> we did not find a meta file - so we try with the next URL in the loop.
	}

	pterm.Info.Printfln("Please specify the base URL of the production server (f.e. github.com).")

	//////////////////// MANUAL ENTRY
	for true {
		// auto-detection did not work; so we need to ask the user for the hostname.
		baseUrlCandidate := textinput.Exec("Base URL")

		baseUrlCandidate = strings.TrimSpace(baseUrlCandidate)
		originalBaseUrlCandidate := strings.TrimSuffix(baseUrlCandidate, "/")
		// the user can enter the URL with or without http/https prefix, and with or without www. prefix.
		// we try to make it as convenient as possible here for the user :)
		// => we remove http:// and https://, so that we can try it with or without https then.
		baseUrlCandidate = strings.TrimPrefix(baseUrlCandidate, "http://")
		baseUrlCandidate = strings.TrimPrefix(baseUrlCandidate, "https://")

		baseUrlCandidates := [...]string{
			originalBaseUrlCandidate,
			"https://" + baseUrlCandidate,
			"http://" + baseUrlCandidate,
			"https://www." + baseUrlCandidate,
			"http://www." + baseUrlCandidate,
		}

		for _, candidate := range baseUrlCandidates {
			_, err = url.ParseRequestURI(candidate)
			if err != nil {
				pterm.Debug.Printfln("Skipping candidate host %s because it is not valid.", candidate)
				continue
			}

			rs.BaseUrl(candidate)
			err = rs.DoesMetaFileExistOnServer()
			if err == nil {
				// we found the meta file; so we are done.
				pterm.Success.Printfln("Found correct base URL at %s.", candidate)

				updateSyncoYmlFile := boolselect.Exec("Update .synco.yml file?", true)

				if updateSyncoYmlFile {
					pterm.Debug.Printfln("Updating %s file with host %s", config.SyncoYamlFile, candidate)
					syncoConfig.Hosts = append(syncoConfig.Hosts, config.SyncoHostConfig{
						BaseUrl: candidate,
					})
					err := config.WriteToFile(syncoConfig)
					if err != nil {
						pterm.Fatal.Printfln("could not update file %s: %s", config.SyncoYamlFile, err)
					}

					pterm.Success.Printfln("Updated %s", config.SyncoYamlFile)
				}

				// NOTE: the receiveSession is already updated; so we do not need to update anything.
				return nil
			}

			if !errors.Is(err, receive.ErrMetaFileNotFound) {
				// we got an unexpected error -> bubble up
				return err
			}
			// receive.ErrMetaFileNotFound -> we did not find a meta file - so we try with the next URL in the loop.
		}

		// when we end up here, we were not successful with finding the base URL. we need to ask again.
		pterm.Warning.Printfln("We could not find the file %s/%s. Please supply a new base URL.", baseUrlCandidate, rs.MetaUrlRelativeToBaseUrl())
	}

	// never reached, because of "while true" loop above.
	return nil
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

	i := 0
	skipped := 0
	// download file.
	progress, _ := pterm.DefaultProgressbar.WithTotal(int(fileSet.PublicFiles.SizeBytes)).Start()

	for fileName, fileDefinition := range publicFilesIndex {
		i++

		// Check for changes of the files (based on size and modification times)
		fileStat, err := receiveSession.StatInWorkDir(fileName)
		if err == nil {
			if fileStat.Size() == fileDefinition.SizeBytes && fileStat.ModTime().Unix() == fileDefinition.MTime {
				// file exists; and exists with same size and modification time. We can skip the download.
				pterm.Debug.Printfln("Ignoring file %s, because it exists already with same size and modification timestamp", fileName)
				progress.Add(int(fileDefinition.SizeBytes))
				skipped++
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

		// the default base URL is the transfer session, which is by convention placed INSIDE the web root.
		// so we need to go one level up when finding <BASE>
		// TODO: this is a bit brittle - but for now it works.
		err = receiveSession.DumpFileWithProgressBar(strings.ReplaceAll(fileDefinition.PublicUri, "<BASE>", ".."), fileName, progress)
		if err != nil {
			pterm.Error.Printfln("error on downloading %s to %s: %w - continuing with next file", fileDefinition.PublicUri, fileName, err)
			// continue with next iteration
		} else {
			// set the desired modification time to the server's modification time (for change tracking)
			desiredMtime := time.Unix(fileDefinition.MTime, 0)
			pterm.Debug.Printfln("Setting mtime for %s to %d", fileName, fileDefinition.MTime)
			err = receiveSession.SetMTimeInWorkDir(fileName, desiredMtime)
			if err != nil {
				return err
			}
		}
	}
	pterm.DefaultBasicText.Sprintf("Downloaded %d files (Skipped: %d)", len(publicFilesIndex), skipped)

	return nil
}

func init() {
	ReceiveCmd.Flags().BoolVar(&interactive, "interactive", true, "interactively select which files to download")
}
