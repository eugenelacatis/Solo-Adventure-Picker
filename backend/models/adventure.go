package models

type Adventure struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type,omitempty"`
	Region      string  `json:"region"`
	Scenery     string  `json:"scenery,omitempty"`
	Effort      string  `json:"effort,omitempty"`
	Duration    string  `json:"duration,omitempty"`
	Description string  `json:"description,omitempty"`
	XPValue     int     `json:"xpValue,omitempty"`
	Lat         float64 `json:"lat,omitempty"`
	Lng         float64 `json:"lng,omitempty"`
}
