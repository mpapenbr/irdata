package irdata

//nolint:tagliatelle // external definition
type (
	SeasonList struct {
		Seasons []Season `json:"seasons,omitempty"`
	}
	Season struct {
		SeasonID        int               `json:"season_id,omitempty"`
		SeasonYear      int               `json:"season_year,omitempty"`
		SeasonQuarter   int               `json:"season_quarter,omitempty"`
		SeasonName      string            `json:"season_name,omitempty"`
		SeasonShortName string            `json:"season_short_name,omitempty"`
		TrackTypes      []SeasonTrackType `json:"track_types,omitempty"`
	}
	SeasonTrackType struct {
		TrackType string `json:"track_type,omitempty"`
	}
	SeasonCarType struct {
		CarType string `json:"car_type,omitempty"`
	}
)
