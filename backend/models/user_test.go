package models

import (
	"encoding/json"
	"testing"
)

func TestUser_JSONRoundTrip(t *testing.T) {
	u := User{
		UserId:  "abc-123",
		TotalXp: 250,
		VisitedAdventures: []VisitedEntry{
			{AdventureId: "adv-1", Lat: 37.9235, Lng: -122.5965},
		},
		JournalEntries: []JournalEntry{
			{AdventureId: "adv-1", Text: "Great hike today!"},
		},
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.UserId != "abc-123" {
		t.Errorf("UserId = %q, want %q", decoded.UserId, "abc-123")
	}
	if decoded.TotalXp != 250 {
		t.Errorf("TotalXp = %d, want 250", decoded.TotalXp)
	}
	if len(decoded.VisitedAdventures) != 1 || decoded.VisitedAdventures[0].AdventureId != "adv-1" {
		t.Errorf("VisitedAdventures = %+v, want one entry with AdventureId adv-1", decoded.VisitedAdventures)
	}
	if len(decoded.JournalEntries) != 1 || decoded.JournalEntries[0].Text != "Great hike today!" {
		t.Errorf("JournalEntries = %+v, want one entry with the expected text", decoded.JournalEntries)
	}
}
