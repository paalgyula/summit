/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use: "auth",
	// Short: "A brief description of your command",
}

func init() {
	authCmd.AddCommand(registerCmd())

	rootCmd.AddCommand(authCmd)

	authCmd.PersistentFlags().String("authserver", "127.0.0.1:4999", "Auth server management interface address")
	authCmd.PersistentFlags().String("token", os.Getenv("SUMMIT_AUTH_MANAGEMENT_TOKEN"), "Management API shared token (SUMMIT_AUTH_MANAGEMENT_TOKEN)")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// authCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func registerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register [username] [password] [email]",
		Short: "Registers a user in the auth server",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			addr := cmd.Flags().Lookup("authserver").Value.String()
			token := cmd.Flags().Lookup("token").Value.String()

			client, err := auth.NewManagementClient(addr, token)
			if err != nil {
				panic(err)
			}

			if err := client.Register(args[0], args[1], args[2]); err != nil {
				fmt.Printf("cannot create account: %s\n", err.Error())
				os.Exit(1)
			}

			fmt.Printf("account [%s] has been created\n", args[0])
		},
	}
}
