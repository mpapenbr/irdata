package racelogger

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/mpapenbr/irdata/auth"
	"github.com/mpapenbr/irdata/cmd/config"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

func NewCarDataCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "cardata",
		Short: "fetch car data from iRacing API and prepare it for racelogger",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			prepareCarData()
			return nil
		},
	}

	return &cmd
}

type (
	carDataFetcher struct {
		ir *irdata.IrData
		tm *auth.TokenManager
	}
	//nolint:tagliatelle // external definition
	irCardata struct {
		CarID                   int    `json:"car_id"`
		CarName                 string `json:"car_name"`
		CarNameAbbreviated      string `json:"car_name_abbreviated"`
		HasMultipleDryTireTypes bool   `json:"has_multiple_dry_tire_types"`
		HasRainCapableTireTypes bool   `json:"has_rain_capable_tire_types"`
	}
	raceloggerCarData struct {
		CarID                   int    `json:"carId"`
		Abbrev                  string `json:"abbrev"`
		Name                    string `json:"name"`
		HasRainCapableTireTypes bool   `json:"hasRainCapableTireTypes"`
		HasMultipleDryTireTypes bool   `json:"hasMultipleDryTireTypes"`
	}
)

func prepareCarData() {
	crl, err := newFetchCarData()
	if err != nil {
		log.Error("failed to initialize carDataFetcher", log.ErrorField(err))
		return
	}
	crl.readData()
}

func newFetchCarData() (*carDataFetcher, error) {
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
	return &carDataFetcher{ir: ir, tm: tm}, nil
}

func (c *carDataFetcher) readData() {
	cd, err := c.ir.Get("/data/car/get")
	if err != nil {
		log.Error("API call failed", log.ErrorField(err))
		return
	}
	log.Info("fetched current season data",
		log.Int("data-size", len(cd)))
	writeToFile("tmp/cardata.json", cd)
	irCarData := make([]irCardata, 0)
	if err = json.Unmarshal(cd, &irCarData); err != nil {
		log.Error("failed to unmarshal car data", log.ErrorField(err))
		return
	}
	raceloggerData := make([]raceloggerCarData, len(irCarData))
	for i := range irCarData {
		c := irCarData[i]
		raceloggerData[i] = raceloggerCarData{
			CarID:                   c.CarID,
			Name:                    c.CarName,
			Abbrev:                  c.CarNameAbbreviated,
			HasMultipleDryTireTypes: c.HasMultipleDryTireTypes,
			HasRainCapableTireTypes: c.HasRainCapableTireTypes,
		}
	}
	data, err := json.MarshalIndent(raceloggerData, "", "  ")
	if err != nil {
		log.Error("failed to marshal racelogger car data", log.ErrorField(err))
		return
	}
	writeToFile("tmp/racelogger_cardata.json", data)
}

func writeToFile(filename string, data []byte) {
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		log.Error("failed to write data to file",
			log.String("filename", filename),
			log.ErrorField(err))
	}
}
