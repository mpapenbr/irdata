package auth

import (
	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/auth"
	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/log"
)

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
	if config.IrAuthConfig.AuthFile != "" {
		log.Debug("auth file path provided",
			log.String("auth-file", config.IrAuthConfig.AuthFile))
	}
	tm, err := auth.NewTokenManager(auth.WithAuthConfig(&config.IrAuthConfig))
	if err != nil {
		log.Error("failed to create token manager", log.ErrorField(err))
		return
	}
	if err := tm.Login(); err != nil {
		log.Error("failed to login", log.ErrorField(err))
		return
	}
	log.Info("successfully logged in to iRacing")
}
