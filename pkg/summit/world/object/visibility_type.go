package object

// VisibilityDistanceType represents the tier of visibility distance for an object.
// Objects with Large/Gigantic overrides are stored in the far-visible grid container.
// Mirrors AzerothCore's VisibilityDistanceType enum in ObjectDefines.h.
type VisibilityDistanceType uint8

const (
	VisibilityDistanceNormal   VisibilityDistanceType = iota // 100y — default
	VisibilityDistanceTiny                                   // 25y — small caves
	VisibilityDistanceSmall                                  // 50y — tight spaces
	VisibilityDistanceLarge                                  // 200y — far-visible creatures
	VisibilityDistanceGigantic                               // 400y — world bosses
	VisibilityDistanceInfinite                               // 533y — zone-wide (grid-sized)
	VisibilityDistanceMax                                    // sentinel
)

// VisibilityDistances maps VisibilityDistanceType to yard values.
// Matches AzerothCore's constexpr float VisibilityDistances[] in Object.cpp.
var VisibilityDistances = [VisibilityDistanceMax]float64{
	VisibilityDistanceNormal:   100.0,
	VisibilityDistanceTiny:     25.0,
	VisibilityDistanceSmall:    50.0,
	VisibilityDistanceLarge:    200.0,
	VisibilityDistanceGigantic: 400.0,
	VisibilityDistanceInfinite: 533.0,
}

// IsFarVisible returns true if this override type uses the far-visible grid container.
// Matches AzerothCore's WorldObject::IsFarVisible.
func (vd VisibilityDistanceType) IsFarVisible() bool {
	return vd == VisibilityDistanceLarge || vd == VisibilityDistanceGigantic
}
