package util

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// initConfig reads in config file and ENV variables if set.
func InitConfig(cfgFile *string) func() {
	return func() {
		if *cfgFile != "" {
			// Use config file from the flag.
			viper.SetConfigFile(*cfgFile)
		} else {
			// Find home directory.
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)

			// Search config in home directory with name ".synco" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".synco")
		}

		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
