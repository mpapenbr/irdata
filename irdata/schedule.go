package irdata

//nolint:tagliatelle // external definition
type (
	ScheduleResponse struct {
		Schedules []Schedule `json:"schedules,omitempty"`
	}
	Schedule struct {
		SeasonID     int  `json:"season_id,omitempty"`
		QualAttached bool `json:"qual_attached,omitempty"`
		RaceWeekNum  int  `json:"race_week_num,omitempty"`
	}
)
