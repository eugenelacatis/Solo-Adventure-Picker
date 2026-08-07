package services

// levelThresholds[i] is the minimum totalXp required to be level i+2
// (level 1 starts at 0 XP and isn't listed).
var levelThresholds = []int{100, 250}

// LevelForXp derives a user's level from their accumulated XP.
func LevelForXp(xp int) int {
	level := 1
	for _, threshold := range levelThresholds {
		if xp >= threshold {
			level++
		}
	}
	return level
}
