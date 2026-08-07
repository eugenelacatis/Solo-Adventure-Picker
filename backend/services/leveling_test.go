package services

import "testing"

func TestLevelForXp(t *testing.T) {
	cases := []struct {
		xp    int
		level int
	}{
		{xp: 0, level: 1},
		{xp: 99, level: 1},
		{xp: 100, level: 2},
		{xp: 249, level: 2},
		{xp: 250, level: 3},
		{xp: 10000, level: 3},
	}

	for _, c := range cases {
		got := LevelForXp(c.xp)
		if got != c.level {
			t.Errorf("LevelForXp(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}
