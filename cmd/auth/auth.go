package auth

import (
	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/log"
)

//nolint:errcheck,lll // by design
func NewLoginCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "login",
		Short: "login to iRacing and save auth info to a file",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			doLogin()
			return nil
		},
	}
	return &cmd
}

func doLogin() {
	log.Debug("Logging in to iRacing...")
}
