package models

import "time"

type VisitedEntry struct {
	AdventureId string    `json:"adventureId"`
	Name        string    `json:"name,omitempty"`
	Region      string    `json:"region,omitempty"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	CreatedAt   time.Time `json:"createdAt"`
}

type JournalEntry struct {
	AdventureId string `json:"adventureId"`
	Text        string `json:"text"`
}

type JournalEntryView struct {
	AdventureId   string    `json:"adventureId"`
	AdventureName string    `json:"adventureName,omitempty"`
	Text          string    `json:"text"`
	CreatedAt     time.Time `json:"createdAt"`
}

type User struct {
	ID                int64          `json:"id"`
	UserId            string         `json:"userId"`
	TotalXp           int            `json:"totalXp"`
	VisitedAdventures []VisitedEntry `json:"visitedAdventures"`
	JournalEntries    []JournalEntry `json:"journalEntries"`
}
