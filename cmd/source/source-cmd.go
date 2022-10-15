package source

import (
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/pkg/framework_detection"
	"github.com/spf13/cobra"
)

var SourceCmd = &cobra.Command{
	Use:     "source",
	Short:   "Wizard to be executed in source",
	Long:    `...`,
	Example: `synco source`,
	// Uncomment the following lines if your bare application has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		progressbar, err := pterm.DefaultProgressbar.WithTotal(3).Start()
		if err != nil {
			return err
		}
		pterm.DefaultBasicText.Println("Detecting Frameworks")

		frameworkDetector := framework_detection.NewFrameworkDetector()
		frameworkDetector.Run()

		pterm.DefaultBasicText.Println("Hallo")

		progressbar.Increment()
		return nil
	},
}
