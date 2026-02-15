package populate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/util"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

var (
	year    []int
	quarter []int
)

func NewPopulateSeriesCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "series",
		Short: "populate series information from iRacing",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			populateSeries()
			return nil
		},
	}
	cmd.PersistentFlags().IntSliceVar(&year, "year",
		[]int{2026}, "iRacing API year")
	cmd.PersistentFlags().IntSliceVar(&quarter, "quarter",
		[]int{1, 2, 3, 4}, "iRacing API quarter (1-4)")

	return &cmd
}

func populateSeries() {
	app, err := util.InitApp()
	if err != nil {
		log.Error("failed to initialize app", log.ErrorField(err))
		return
	}
	defer app.Close()
	if len(quarter) == 0 {
		quarter = []int{1, 2, 3, 4}
	}
	for _, y := range year {
		for _, q := range quarter {
			var data []byte
			var err error
			data, err = app.API.Get(
				fmt.Sprintf("/data/series/season_list?season_year=%d&season_quarter=%d",
					y, q))
			if err != nil {
				log.Error("failed to get current season data", log.ErrorField(err))
				continue
			}
			log.Info("fetched series data for year and quarter",
				log.Int("year", y),
				log.Int("quarter", q),
				log.Int("data-size",
					len(data)))
			os.WriteFile(fmt.Sprintf("tmp/season-%d-%d.json", y, q), data, 0o644)
			var seasons irdata.SeasonList
			if err = json.Unmarshal(data, &seasons); err != nil {
				log.Error("failed to unmarshal season data", log.ErrorField(err))
				continue
			}

			for i := range seasons.Seasons {
				s := seasons.Seasons[i]

				log.Debug("season data",
					log.Int("season_id", s.SeasonID),
					log.Int("season_year", s.SeasonYear),
					log.Int("season_quarter", s.SeasonQuarter),
				)
				data, err = app.API.Get(
					fmt.Sprintf("/data/series/season_schedule?season_id=%d", s.SeasonID))
				if err != nil {
					log.Error("failed to get season schedule data", log.ErrorField(err))
					continue
				}
				os.WriteFile(fmt.Sprintf("tmp/schedule-%d-%d-%d.json", y, q, s.SeasonID), data, 0o644)

			}
			log.Info("season data", log.Int("season_count", len(seasons.Seasons)))

		}
	}
}
