package services

import "testing"

func TestHaversineMiles_SamePoint_ReturnsZero(t *testing.T) {
	d := HaversineMiles(37.8651, -119.5383, 37.8651, -119.5383)
	if d != 0 {
		t.Errorf("HaversineMiles(same point) = %v, want 0", d)
	}
}

func TestHaversineMiles_KnownDistance_MatchesExpected(t *testing.T) {
	// Great-circle distance between these two SF coordinates is ~4.27 miles
	// (independently verified); the window allows for minor floating-point variance
	// without pinning an exact value.
	d := HaversineMiles(37.7793, -122.4193, 37.8199, -122.4783)
	if d < 4.0 || d > 4.5 {
		t.Errorf("HaversineMiles(SF coordinates) = %v, want ~4.27 miles", d)
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
