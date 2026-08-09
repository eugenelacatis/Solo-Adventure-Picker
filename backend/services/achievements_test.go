package services

import (
	"encoding/json"
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

// TestComputeAchievements_NoActivity_EncodesAsEmptyArrayNotNull guards
// against a real bug: a nil Go slice JSON-encodes as `null`, and the
// frontend HUD calls .length on the decoded /achievements response
// expecting an array. A brand-new user (no visits, no journal entries) hit
// exactly this path and crashed the whole page on signup.
func TestComputeAchievements_NoActivity_EncodesAsEmptyArrayNotNull(t *testing.T) {
	got := ComputeAchievements(0, 0)

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal achievements: %v", err)
	}
	if string(encoded) != "[]" {
		t.Errorf("json.Marshal(ComputeAchievements(0, 0)) = %s, want [] (not null)", encoded)
	}
}
