package cmd

import (
	"github.com/pterm/pterm"
	"github.com/repeale/fp-go"
	"github.com/sandstorm/synco/pkg/common/dto"
	"github.com/sandstorm/synco/pkg/receive"
	"github.com/spf13/cobra"
	"os"
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
	return receiveSession.DumpFileWithProgressBar(fileSet.MysqlDump.FileName, fileSet.Name)
}

func init() {
	ReceiveCmd.Flags().BoolVar(&interactive, "interactive", true, "identifier for the decryption")
}
