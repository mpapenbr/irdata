package series

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/auth"
	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

var (
	year    int
	quarter int
)

func NewSeriesCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "series",
		Short: "lookup series information from iRacing",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			querySeries()
			return nil
		},
	}
	cmd.PersistentFlags().IntVar(&year, "year", 2026, "iRacing API year")
	cmd.PersistentFlags().IntVar(&quarter, "quarter", 0, "iRacing API quarter (1-4)")

	return &cmd
}

func querySeries() {
	tm, err := auth.NewTokenManager(auth.WithAuthConfig(&config.IrAuthConfig))
	if err != nil {
		log.Error("failed to create token manager", log.ErrorField(err))
		return
	}
	if err := tm.Login(); err != nil {
		log.Error("failed to login", log.ErrorField(err))
		return
	}
	ir, err := irdata.NewIrData(
		irdata.WithTokenProvider(tm.GetAccessToken),
	)
	if err != nil {
		log.Error("failed to create iRData instance", log.ErrorField(err))
		return
	}
	data, err := ir.Get(fmt.Sprintf("/data/series/season_list?season_year=%d&season_quarter=%d", year, quarter))
	// data, err := ir.Get(fmt.Sprintf("/data/series/season_list"))
	if err != nil {
		log.Error("failed to get current season data", log.ErrorField(err))
		return
	}
	_ = data
	// fmt.Print(string(data))
	// var lookupResponse []irdata.LookupResponse
	// if err := json.Unmarshal(data, &lookupResponse); err != nil {
	// 	log.Error("failed to unmarshal current season data", log.ErrorField(err))
	// 	return
	// }
	// log.Info("current season data", log.Any("data", lookupResponse))
}
