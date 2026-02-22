package irdata

//nolint:tagliatelle // external definition
type (
	// collects all results for a race week
	SeasonResultsResponse struct {
		SeasonID    int            `json:"season_id,omitempty"`
		RaceWeekNum int            `json:"race_week_num,omitempty"`
		ResultsList []SeasonResult `json:"results_list,omitempty"`
	}
	// contains info to retrieve the event results via subsession id
	SeasonResult struct {
		SessionID       int  `json:"session_id,omitempty"`
		SubsessionID    int  `json:"subsession_id,omitempty"`
		OfficialSession bool `json:"official_session,omitempty"`
	}
	// the data for an event result, retrieved via subsession id
	// Note: contains only those attributes we're intereted in
	EventResult struct {
		SessionID         int                  `json:"session_id,omitempty"`
		SubsessionID      int                  `json:"subsession_id,omitempty"`
		CornersPerLap     int                  `json:"corners_per_lap,omitempty"`
		OfficialSession   bool                 `json:"official_session,omitempty"`
		SessionResults    []EventSessionResult `json:"session_results,omitempty"`
		EventLapsComplete int                  `json:"event_laps_complete,omitempty"`
	}

	EventSessionResult struct {
		Results        []EventSessionResultEntry `json:"results,omitempty"`
		SimSessionName string                    `json:"simsession_name,omitempty"`
		SimSessionType int                       `json:"simsession_type,omitempty"`
	}
	EventSessionResultEntry struct {
		CustID          int     `json:"cust_id,omitempty"`
		DisplayName     string  `json:"display_name,omitempty"`
		Incidents       int     `json:"incidents,omitempty"`
		LapsComplete    int     `json:"laps_complete,omitempty"`
		NewCPI          float64 `json:"new_cpi,omitempty"`
		NewLicenseLevel int     `json:"new_license_level,omitempty"`
		NewSubLevel     int     `json:"new_sub_level,omitempty"`
		OldCPI          float64 `json:"old_cpi,omitempty"`
		OldLicenseLevel int     `json:"old_license_level,omitempty"`
		OldSubLevel     int     `json:"old_sub_level,omitempty"`
	}
)
