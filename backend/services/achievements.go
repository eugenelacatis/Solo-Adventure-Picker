package services

type Achievement struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// ComputeAchievements derives which milestones a user has unlocked from their
// visited-adventure and journal-entry counts. Computed on demand; not persisted.
func ComputeAchievements(visitedCount, journalCount int) []Achievement {
	var achievements []Achievement

	if visitedCount >= 1 {
		achievements = append(achievements, Achievement{Id: "first-adventure", Name: "First Adventure"})
	}
	if visitedCount >= 5 {
		achievements = append(achievements, Achievement{Id: "five-adventures", Name: "Five Adventures"})
	}
	if journalCount >= 1 {
		achievements = append(achievements, Achievement{Id: "first-journal", Name: "First Journal Entry"})
	}

	return achievements
}
