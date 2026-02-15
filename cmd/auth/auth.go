package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "auth",
		Short: "commands related to authentication",
		Long:  ``,
	}

	cmd.AddCommand(NewLoginCommand())
	return &cmd
}
