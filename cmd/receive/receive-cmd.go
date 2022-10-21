package receive

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/receive"
	"github.com/spf13/cobra"
	"os"
)

var CommandDeclaration = &cobra.Command{
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

		frameworkName, err := receiveSession.FetchFrameworkName()
		pterm.PrintOnErrorf("Error fetching framework name: %e", err)
		if err != nil {
			return
		}

		pterm.Success.Printfln("Valid Decryption Key")
		pterm.Success.Printfln("Framework on server: %s", frameworkName)

		//pterm.Error.Printfln("No frameworks could be detected. You can manually create a syncro.yaml config file with %s.", pterm.ThemeDefault.HighlightStyle.Sprint("syncro config init"))
		os.Exit(1)
	},
}
