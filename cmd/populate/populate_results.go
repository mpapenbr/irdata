package populate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/util"
	"github.com/mpapenbr/irdata/log"
)

var inputFile string

func NewPopulateResultsCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "results",
		Short: "populate results information from iRacing",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			populateResults()
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&inputFile, "input-file", "",
		"Input file for results data")

	return &cmd
}

func populateResults() {
	app, err := util.InitApp()
	if err != nil {
		log.Error("failed to initialize app", log.ErrorField(err))
		return
	}
	defer app.Close()
	data, err := os.ReadFile(inputFile)
	if err != nil {
		log.Error("failed to read input file", log.ErrorField(err))
		return
	}

	var results []ResultData
	if err := json.Unmarshal(data, &results); err != nil {
		log.Error("failed to parse results data", log.ErrorField(err))
		return
	}
	log.Info("successfully parsed results data", log.Int("num_results", len(results)))
	for i := range results {
		r := results[i]
		var data []byte
		var err error
		// just races
		data, err = app.API.Get(
			fmt.Sprintf(`
			/data/results/season_results?season_id=%d&race_week_num=%d&event_type=5
			`, r.SeasonID, r.RaceWeekNum))
		if err != nil {
			log.Error("failed to get current season data", log.ErrorField(err))
			continue
		}
		writeToFile(fmt.Sprintf("tmp/results-%d-%d.json", r.SeasonID, r.RaceWeekNum), data)
	}
}
