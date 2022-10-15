package source

import "github.com/spf13/cobra"

var SourceCmd = &cobra.Command{
	Use:     "source",
	Short:   "Wizard to be executed in source",
	Long:    `...`,
	Example: `synco source`,
	// Uncomment the following lines if your bare application has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
