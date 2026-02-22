package populate

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jszwec/csvutil"
	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/util"
	"github.com/mpapenbr/irdata/irdata"
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
	collector := &collectResults{app: app}
	collector.process(results)
}

type (
	collectResults struct {
		app     *util.App
		results []CSVResult
	}
	//nolint:tagliatelle // external definition
	CSVResult struct {
		CustID          int     `json:"cust_id,omitempty"           csv:"cust_id"`
		Name            string  `json:"name,omitempty"              csv:"name"`
		SubsessionID    int     `json:"subsession_id,omitempty"     csv:"subsession_id"`
		LapsComplete    int     `json:"laps_complete,omitempty"     csv:"laps_complete"`
		Incidents       int     `json:"incidents,omitempty"         csv:"incidents"`
		NewCPI          float64 `json:"new_cpi,omitempty"           csv:"new_cpi"`
		NewLicenseLevel int     `json:"new_license_level,omitempty" csv:"new_license_level"`
		NewSubLevel     int     `json:"new_sub_level,omitempty"     csv:"new_sub_level"`
		OldCPI          float64 `json:"old_cpi,omitempty"           csv:"old_cpi"`
		OldLicenseLevel int     `json:"old_license_level,omitempty" csv:"old_license_level"`
		OldSubLevel     int     `json:"old_sub_level,omitempty"     csv:"old_sub_level"`
	}
)

func (c *collectResults) process(results []ResultData) {
	c.results = make([]CSVResult, 0)
	for i := range results {
		r := results[i]
		var data []byte
		var err error
		// just races
		data, err = c.app.API.Get(
			strings.TrimSpace(fmt.Sprintf(`
			/data/results/season_results?season_id=%d&race_week_num=%d&event_type=5
			`, r.SeasonID, r.RaceWeekNum)))
		if err != nil {
			log.Error("failed to get current season data", log.ErrorField(err))
			continue
		}
		var seasonResults irdata.SeasonResultsResponse
		err = json.Unmarshal(data, &seasonResults)
		if err != nil {
			log.Error("failed to parse season results data", log.ErrorField(err))
			continue
		}
		c.collectRaceResults(&seasonResults)
	}
	err := c.writeResults()
	if err != nil {
		log.Error("failed to write results", log.ErrorField(err))
	}
}

func (c *collectResults) writeResults() error {
	f, err := os.Create("tmp/00-result.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)

	enc := csvutil.NewEncoder(w)
	if err := enc.EncodeHeader(CSVResult{}); err != nil { // writes header based on tags
		return err
	}

	for i := range c.results {
		if err := enc.Encode(c.results[i]); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}

//nolint:whitespace // editor/linter issue
func (c *collectResults) collectRaceResults(
	seasonResults *irdata.SeasonResultsResponse,
) {
	for i := range seasonResults.ResultsList {
		r := seasonResults.ResultsList[i]
		data, err := c.app.API.Get(
			strings.TrimSpace(fmt.Sprintf(`
			/data/results/get?subsession_id=%d
			`, r.SubsessionID)))
		if err != nil {
			log.Error("failed to get event result data", log.ErrorField(err))
			continue
		}
		var eventResult irdata.EventResult
		err = json.Unmarshal(data, &eventResult)
		if err != nil {
			log.Error("failed to parse event result data", log.ErrorField(err))
			continue
		}
		c.processEventResult(&eventResult)
	}
}

//nolint:funlen,whitespace // much to do here
func (c *collectResults) processEventResult(
	er *irdata.EventResult,
) {
	log.Debug("processing event result",
		log.Int("session_id", er.SessionID),
		log.Int("subsession_id", er.SubsessionID),
		log.Int("corners_per_lap", er.CornersPerLap),
		log.Int("num_sessions", len(er.SessionResults)),
	)
	if !er.OfficialSession {
		return
	}
	custIDs := c.checkPreRaceIncidents(er)
	for i := range er.SessionResults {
		// TODO: collect incidents from practice
		sr := er.SessionResults[i]
		if sr.SimSessionType != 6 {
			continue
		}
		for j := range sr.Results {
			r := sr.Results[j]
			if r.LapsComplete != er.EventLapsComplete {
				log.Debug("skipping session result entry with incomplete laps",
					log.Int("laps_complete", r.LapsComplete),
					log.Int("event_laps_complete", er.EventLapsComplete),
				)
				continue
			}
			if _, ok := custIDs[r.CustID]; ok {
				log.Debug("skipping session result entry with pre-race incidents",
					log.Int("cust_id", r.CustID),
					log.String("display_name", r.DisplayName),
				)
				continue
			}
			c.results = append(c.results, CSVResult{
				CustID:          r.CustID,
				Name:            r.DisplayName,
				SubsessionID:    er.SubsessionID,
				Incidents:       r.Incidents,
				LapsComplete:    r.LapsComplete,
				NewCPI:          r.NewCPI,
				NewLicenseLevel: r.NewLicenseLevel,
				NewSubLevel:     r.NewSubLevel,
				OldCPI:          r.OldCPI,
				OldLicenseLevel: r.OldLicenseLevel,
				OldSubLevel:     r.OldSubLevel,
			})
		}
	}
}

//nolint:whitespace // editor/linter issue
func (c *collectResults) checkPreRaceIncidents(
	er *irdata.EventResult,
) map[int]struct{} {
	ret := make(map[int]struct{})
	for i := range er.SessionResults {
		sr := er.SessionResults[i]
		if sr.SimSessionType == 6 {
			continue
		}
		for j := range sr.Results {
			r := sr.Results[j]
			if r.Incidents > 0 {
				log.Debug("pre-race incident detected",
					log.Int("cust_id", r.CustID),
					log.String("display_name", r.DisplayName),
					log.Int("subsession_id", er.SubsessionID),
					log.Int("sim_session_type", sr.SimSessionType),
					log.Int("incidents", r.Incidents),
				)
				ret[r.CustID] = struct{}{}
			}
		}
	}
	return ret
}
