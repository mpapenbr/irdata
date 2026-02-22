package check

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/auth"
	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

func NewCheckRateLimitCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "rate-limit",
		Short: "commands related to checking rate limits in irdata",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkRateLimits()
			return nil
		},
	}
	cmd.Flags().DurationVar(&rate, "rate", 1*time.Second,
		"duration between API calls to check rate limits")

	return &cmd
}

var rate time.Duration = 1 * time.Second

type checkRateLimit struct {
	ir *irdata.IrData
	tm *auth.TokenManager
}

func checkRateLimits() {
	crl, err := newCheckRateLimit()
	if err != nil {
		log.Error("failed to initialize checkRateLimit", log.ErrorField(err))
		return
	}
	crl.check(rate)
}

func newCheckRateLimit() (*checkRateLimit, error) {
	tm, tmErr := auth.NewTokenManager(auth.WithAuthConfig(&config.IrAuthConfig))
	if tmErr != nil {
		log.Error("failed to create token manager", log.ErrorField(tmErr))
		return nil, tmErr
	}
	if loginErr := tm.Login(); loginErr != nil {
		log.Error("failed to login", log.ErrorField(loginErr))
		return nil, loginErr
	}

	ir, irErr := irdata.NewIrData(
		irdata.WithTokenProvider(tm.GetAccessToken),
	)
	if irErr != nil {
		log.Error("failed to create iRData instance", log.ErrorField(irErr))
		return nil, irErr
	}
	return &checkRateLimit{ir: ir, tm: tm}, nil
}

func (c *checkRateLimit) check(pause time.Duration) {
	t := time.NewTicker(pause)
	defer t.Stop()
	for range t.C {
		_, err := c.ir.Get("/data/lookup/current_season")
		if err != nil {
			log.Error("API call failed", log.ErrorField(err))
			return
		}
	}
}
