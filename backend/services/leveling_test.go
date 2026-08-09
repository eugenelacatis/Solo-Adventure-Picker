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
		{xp: 299, level: 2},
		{xp: 300, level: 3},
		{xp: 599, level: 3},
		{xp: 600, level: 4},
		{xp: 10000, level: 14},
	}

	for _, c := range cases {
		got := LevelForXp(c.xp)
		if got != c.level {
			t.Errorf("LevelForXp(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}

func TestNextLevelXp(t *testing.T) {
	cases := []struct {
		xp   int
		want int
	}{
		{xp: 0, want: 100},
		{xp: 99, want: 100},
		{xp: 100, want: 300},
		{xp: 299, want: 300},
		{xp: 300, want: 600},
	}

	for _, c := range cases {
		got := NextLevelXp(c.xp)
		if got != c.want {
			t.Errorf("NextLevelXp(%d) = %d, want %d", c.xp, got, c.want)
		}
	}
}
