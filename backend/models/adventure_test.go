package models

import (
	"encoding/json"
	"testing"
)

func TestAdventure_JSONRoundTrip_IncludesLatLng(t *testing.T) {
	adv := Adventure{
		Name:   "Mount Tamalpais",
		Region: "bay-area",
		Lat:    37.9235,
		Lng:    -122.5965,
	}

	data, err := json.Marshal(adv)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded Adventure
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.Lat != 37.9235 {
		t.Errorf("Lat = %v, want 37.9235", decoded.Lat)
	}
	if decoded.Lng != -122.5965 {
		t.Errorf("Lng = %v, want -122.5965", decoded.Lng)
	}
}
