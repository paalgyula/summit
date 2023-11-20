package main

import (
	"github.com/paalgyula/summit/cmd/summitctl/cmd"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("management")    // name of config file (without extension)
	viper.SetConfigType("yaml")          // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.summit") // call multiple times to add many search paths
	viper.AddConfigPath(".")             // optionally look for config in the working directory
	err := viper.ReadInConfig()          // Find and read the config file
	if err != nil {                      // Handle errors reading the config file
		viper.SafeWriteConfig()

		// panic(fmt.Errorf("fatal error config file: %w", err))
	}

	cmd.Execute()
}
