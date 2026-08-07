package services

import (
	"reflect"
	"testing"
)

func TestComputeAchievements(t *testing.T) {
	cases := []struct {
		name           string
		visitedCount   int
		journalCount   int
		wantAchieveIds []string
	}{
		{
			name:           "no activity yet",
			visitedCount:   0,
			journalCount:   0,
			wantAchieveIds: nil,
		},
		{
			name:           "first hike unlocked",
			visitedCount:   1,
			journalCount:   0,
			wantAchieveIds: []string{"first-adventure"},
		},
		{
			name:           "five hikes unlocked",
			visitedCount:   5,
			journalCount:   0,
			wantAchieveIds: []string{"first-adventure", "five-adventures"},
		},
		{
			name:           "first journal unlocked",
			visitedCount:   0,
			journalCount:   1,
			wantAchieveIds: []string{"first-journal"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ComputeAchievements(c.visitedCount, c.journalCount)
			var gotIds []string
			for _, a := range got {
				gotIds = append(gotIds, a.Id)
			}
			if !reflect.DeepEqual(gotIds, c.wantAchieveIds) {
				t.Errorf("ComputeAchievements(%d, %d) ids = %v, want %v", c.visitedCount, c.journalCount, gotIds, c.wantAchieveIds)
			}
		})
	}
}
