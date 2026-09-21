package main

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/paalgyula/summit/cmd/summitctl/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func main() {
	// godotenv loads .env into os.Environ — viper picks it up via AutomaticEnv
	_ = godotenv.Load()

	viper.SetConfigName("management")    // name of config file (without extension)
	viper.SetConfigType("yaml")          // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("$HOME/.summit") // call multiple times to add many search paths
	viper.AddConfigPath(".")             // optionally look for config in the working directory
	err := viper.ReadInConfig()          // Find and read the config file
	if err != nil {                      // Handle errors reading the config file
		viper.SafeWriteConfig()
	}

	viper.SetEnvPrefix("summit")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.BindEnv("debug", "DEBUG") //nolint:errcheck

	setupLogger()

	cmd.Execute()
}

func setupLogger() {
	debug := viper.GetBool("debug")
	if !debug {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

		return
	}

	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05.000"}
	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.DebugLevel)
}
