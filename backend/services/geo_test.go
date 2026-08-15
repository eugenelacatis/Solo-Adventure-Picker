package services

import "testing"

func TestHaversineMiles_SamePoint_ReturnsZero(t *testing.T) {
	d := HaversineMiles(37.8651, -119.5383, 37.8651, -119.5383)
	if d != 0 {
		t.Errorf("HaversineMiles(same point) = %v, want 0", d)
	}
}

func TestHaversineMiles_KnownDistance_MatchesExpected(t *testing.T) {
	// San Francisco City Hall to Golden Gate Bridge toll plaza, ~4.9 miles.
	d := HaversineMiles(37.7793, -122.4193, 37.8199, -122.4783)
	if d < 4.5 || d > 5.3 {
		t.Errorf("HaversineMiles(SF landmarks) = %v, want ~4.9 miles", d)
	}
}

func TestHaversineMiles_TightRadius_MatchesWithinThreshold(t *testing.T) {
	// ~100 feet apart (well within the 150ft/46m match radius).
	d := HaversineMiles(37.8651, -119.5383, 37.86513, -119.5383)
	const matchRadiusMiles = 150.0 / 5280.0
	if d > matchRadiusMiles {
		t.Errorf("HaversineMiles(~100ft apart) = %v miles, want <= %v (150ft radius)", d, matchRadiusMiles)
	}
}
