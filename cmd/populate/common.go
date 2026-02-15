package populate

import (
	"os"

	"github.com/mpapenbr/irdata/log"
)

type (
	ResultData struct {
		SeasonID      int    `json:"seasonId,omitempty"`
		SeasonYear    int    `json:"seasonYear,omitempty"`
		SeasonQuarter int    `json:"seasonQuarter,omitempty"`
		SeasonName    string `json:"seasonName,omitempty"`
		RaceWeekNum   int    `json:"raceWeekNum,omitempty"`
	}
)

func writeToFile(filename string, data []byte) {
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		log.Error("failed to write data to file",
			log.String("filename", filename),
			log.ErrorField(err))
	}
}
