package irdata

//nolint:tagliatelle // external definition
type (
	// used in results
	Track struct {
		TrackID    int    `json:"track_id,omitempty"`
		TrackName  string `json:"track_name,omitempty"`
		ConfigName string `json:"config_name,omitempty"`
		Category   string `json:"category,omitempty"`
		CategoryID int    `json:"category_id,omitempty"`
	}
)
