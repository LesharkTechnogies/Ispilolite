package utils

import "math"

const (
	EarthRadiusKM = 6371.0088
	EarthRadiusM  = EarthRadiusKM * 1000
)

// CoordinatesValid reports whether lat/lon are valid WGS84 coordinates.
func CoordinatesValid(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && !math.IsInf(lat, 0) && !math.IsInf(lon, 0) &&
		lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// HaversineDistanceKM returns the great-circle distance between two points.
func HaversineDistanceKM(lat1, lon1, lat2, lon2 float64) float64 {
	if !CoordinatesValid(lat1, lon1) || !CoordinatesValid(lat2, lon2) {
		return math.NaN()
	}
	lat1, lat2 = lat1*math.Pi/180, lat2*math.Pi/180
	dLat := lat2 - lat1
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return EarthRadiusKM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// HaversineDistance is kept as a convenient alias and returns kilometres.
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	return HaversineDistanceKM(lat1, lon1, lat2, lon2)
}

// DistanceMeters returns the great-circle distance in metres.
func DistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	return HaversineDistanceKM(lat1, lon1, lat2, lon2) * 1000
}

// IsWithinRadius reports whether two valid points are no farther apart than
// radiusKM. A non-positive radius is never considered a match.
func IsWithinRadius(lat1, lon1, lat2, lon2, radiusKM float64) bool {
	d := HaversineDistanceKM(lat1, lon1, lat2, lon2)
	return radiusKM > 0 && !math.IsNaN(d) && d <= radiusKM
}

// Bearing returns the initial bearing from the first point to the second in
// degrees clockwise from true north, normalized to [0, 360).
func Bearing(lat1, lon1, lat2, lon2 float64) float64 {
	if !CoordinatesValid(lat1, lon1) || !CoordinatesValid(lat2, lon2) {
		return math.NaN()
	}
	phi1, phi2 := lat1*math.Pi/180, lat2*math.Pi/180
	dLon := (lon2 - lon1) * math.Pi / 180
	y := math.Sin(dLon) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLon)
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}
