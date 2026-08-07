package models

type VisitedEntry struct {
	AdventureId string  `json:"adventureId"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

type JournalEntry struct {
	AdventureId string `json:"adventureId"`
	Text        string `json:"text"`
}

type User struct {
	ID                int64          `json:"id"`
	UserId            string         `json:"userId"`
	TotalXp           int            `json:"totalXp"`
	VisitedAdventures []VisitedEntry `json:"visitedAdventures"`
	JournalEntries    []JournalEntry `json:"journalEntries"`
}
