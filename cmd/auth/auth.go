package auth

import (
	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/config"
)

func NewAuthCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "auth",
		Short: "commands related to authentication",
		Long:  ``,
	}
	cmd.PersistentFlags().StringVar(&config.IrAuthConfig.ClientID,
		"client-id", "", "iRacing API client ID")
	cmd.PersistentFlags().StringVar(&config.IrAuthConfig.ClientSecret,
		"client-secret", "", "iRacing API client secret")
	cmd.PersistentFlags().StringVar(&config.IrAuthConfig.Username,
		"username", "", "iRacing username")
	cmd.PersistentFlags().StringVar(&config.IrAuthConfig.Password,
		"password", "", "iRacing password")
	cmd.PersistentFlags().StringVar(&config.IrAuthConfig.AuthFile,
		"auth-file", "", "temp. auth file")
	cmd.AddCommand(NewLoginCommand())
	return &cmd
}
