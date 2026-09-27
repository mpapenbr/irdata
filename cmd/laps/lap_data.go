package laps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/cmd/util"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
	gUtil "github.com/mpapenbr/irdata/util"
)

func NewLapEventsCommand() *cobra.Command {
	subsessionID := 0
	sessionNumber := 0
	cmd := cobra.Command{
		Use:   "events",
		Short: "collect lap data event of a session from iRacing",
		Long: `
collect lap data event of a session from iRacing.

The race sessions of the requested iRacing session will be processed to
collect lap data events. For each driver in the session,
incidents such as off tracks, contacts, car contacts, and lost control will be recorded.
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := collectLapDataCommand{
				subsessionID:  subsessionID,
				sessionNumber: sessionNumber,
			}
			if err := runner.run(cmd.Context()); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().
		IntVar(&subsessionID, "subsession-id", 0,
			"iRacing subsession ID to collect splits data for")

	return &cmd
}

type (
	collectLapDataCommand struct {
		subsessionID  int
		sessionNumber int
		app           *util.App
		logger        *log.Logger
	}
	incidentData struct {
		LapNumber int
		Events    []string
	}
	sessionIncidents struct {
		RefID int
		Name  string

		Incidents []*incidentData
	}
	siTry []*sessionIncidents
)

func (si siTry) Output(out io.Writer) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "RefID\tName\tOfftracks\tContacts\tCar Contacts\tLost Control")
	for _, entry := range si {
		offtracks := 0
		contacts := 0
		carContacts := 0
		lostControl := 0
		for _, inc := range entry.Incidents {
			for _, ev := range inc.Events {
				if strings.EqualFold(ev, "off track") {
					offtracks++
				}
				if strings.EqualFold(ev, "contact") {
					contacts++
				}
				if strings.EqualFold(ev, "car contact") {
					carContacts++
				}
				if strings.EqualFold(ev, "lost control") {
					lostControl++
				}
			}
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%d\t%d\n",
			entry.RefID, entry.Name, offtracks, contacts, carContacts, lostControl)
	}
	tw.Flush()
}

func (c *collectLapDataCommand) run(ctx context.Context) error {
	app, err := util.InitApp()
	if err != nil {
		log.Error("failed to initialize app", log.ErrorField(err))
		return err
	}
	defer app.Close()
	c.app = app
	c.logger = log.GetFromContext(ctx)
	c.logger.Info("starting lap data collection",
		log.Int("subsession_id", c.subsessionID))
	c.logger.Debug("get event results")
	eventResult, err := c.getEventResult()
	if err != nil {
		log.Error("failed to get event result", log.ErrorField(err))
		return err
	}
	for i := range eventResult.SessionResults {
		sResults := eventResult.SessionResults[i]
		if sResults.SimSessionTypeName == "Race" {

			incidents := c.incCollector(
				eventResult.MaxTeamDrivers > 1,
				sResults.SimSessionNumber,
				sResults.Results,
			)

			fmt.Fprintf(os.Stdout, "Session %s\n", sResults.SimSessionName)
			incidents.Output(os.Stdout)
		}
	}

	return nil
}

//nolint:whitespace // editor/linter issue
func (c *collectLapDataCommand) incCollector(
	isTeamRace bool,
	sessionNumber int,
	resultEntries []irdata.EventSessionResultEntry,
) siTry {
	result := siTry{}
	mu := sync.Mutex{}

	worker := gUtil.NewWorker[irdata.EventSessionResultEntry, sessionIncidents](
		func(entry irdata.EventSessionResultEntry) (sessionIncidents, error) {
			incs, err := c.collectLapData(isTeamRace, sessionNumber, &entry)
			if err != nil {
				log.Error("failed to collect lap data", log.ErrorField(err))
				return sessionIncidents{}, err
			}
			id := entry.CustID
			if isTeamRace {
				id = entry.TeamID
			}
			return sessionIncidents{
				RefID:     id,
				Name:      entry.DisplayName,
				Incidents: incs,
			}, nil
		}, gUtil.WithResultCallback(func(idx int, si sessionIncidents, err error) {
			if err != nil {
				return
			}
			mu.Lock()
			result = append(result, &si)
			mu.Unlock()
		}),
		gUtil.WithNumWorker[sessionIncidents](config.NumWorkers),
	)
	worker.Process(resultEntries)
	return result
}

//nolint:whitespace // editor/linter issue
func (c *collectLapDataCommand) collectLapData(
	isTeamRace bool,
	sessionNumber int, resultEntry *irdata.EventSessionResultEntry,
) ([]*incidentData, error) {
	v := url.Values{}
	v.Set("subsession_id", fmt.Sprintf("%d", c.subsessionID))
	v.Set("simsession_number", fmt.Sprintf("%d", sessionNumber))

	if isTeamRace {
		v.Set("team_id", fmt.Sprintf("%d", resultEntry.TeamID))
	} else {
		v.Set("cust_id", fmt.Sprintf("%d", resultEntry.CustID))
	}
	data, err := c.app.API.Get(
		strings.TrimSpace(fmt.Sprintf(`/data/results/lap_data?%s`, v.Encode())),
	)
	if err != nil {
		log.Error("failed to get lap data", log.ErrorField(err))
		return nil, err
	}

	var chunkedData irdata.ChunkData[irdata.LapData]
	err = json.Unmarshal(data, &chunkedData)
	if err != nil {
		log.Error("failed to parse lap data", log.ErrorField(err))
		return nil, err
	}
	var incidents []*incidentData
	for i := range chunkedData.Data {
		lapData := chunkedData.Data[i]
		if len(lapData.LapEvents) > 0 {
			c.logger.Debug("lap data has events",
				log.Int("LapNo", lapData.LapNumber),
				log.Any("events", lapData.LapEvents))
			incidents = append(incidents, &incidentData{
				LapNumber: lapData.LapNumber,
				Events:    lapData.LapEvents,
			})
		}
	}
	return incidents, nil
}

//nolint:whitespace // editor/linter issue
func (c *collectLapDataCommand) getEventResult() (
	*irdata.EventResult, error,
) {
	data, err := c.app.API.Get(
		strings.TrimSpace(fmt.Sprintf(`
			/data/results/get?subsession_id=%d
			`, c.subsessionID)),
	)
	if err != nil {
		log.Error("failed to get event result data", log.ErrorField(err))
		return nil, err
	}
	var eventResult irdata.EventResult
	err = json.Unmarshal(data, &eventResult)
	if err != nil {
		log.Error("failed to parse event result data", log.ErrorField(err))
		return nil, err
	}
	return &eventResult, nil
}
