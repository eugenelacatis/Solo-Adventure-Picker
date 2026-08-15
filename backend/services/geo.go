package services

import "math"

const earthRadiusMiles = 3958.8

// HaversineMiles returns the great-circle distance in miles between two
// lat/lng points, using the haversine formula. Shared by trail proximity
// matching (POST /trail) and, later, travel-time/distance-sort features —
// written once here so both consume the same math.
func HaversineMiles(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }

	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	rlat1 := toRad(lat1)
	rlat2 := toRad(lat2)

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rlat1)*math.Cos(rlat2)*math.Sin(dLng/2)*math.Sin(dLng/2)

	return 2 * earthRadiusMiles * math.Asin(math.Sqrt(h))
}
