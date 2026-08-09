package services

// xpForLevel returns the total accumulated XP required to reach level.
// Level 1 starts at 0 XP. Each subsequent level costs progressively more
// (a triangular curve: level 2 at 100, level 3 at 300, level 4 at 600, ...)
// so leveling stays achievable early on but keeps scaling as play continues,
// instead of capping out the way the old fixed two-threshold list did.
func xpForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	n := level - 1
	return 100 * n * (n + 1) / 2
}

// LevelForXp derives a user's level from their accumulated XP.
func LevelForXp(xp int) int {
	level := 1
	for xp >= xpForLevel(level+1) {
		level++
	}
	return level
}

// NextLevelXp returns the total XP needed to reach the next level above xp's
// current level, for HUD progress-bar display.
func NextLevelXp(xp int) int {
	return xpForLevel(LevelForXp(xp) + 1)
}
