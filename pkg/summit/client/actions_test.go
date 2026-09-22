package client

import (
	"math"
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
)

func TestOrientationTo(t *testing.T) {
	from := player.WorldLocation{X: 0, Y: 0}

	cases := []struct {
		name string
		to   player.WorldLocation
		want float32
	}{
		{"east", player.WorldLocation{X: 10, Y: 0}, 0},
		{"north", player.WorldLocation{X: 0, Y: 10}, float32(math.Pi / 2)},
		{"west", player.WorldLocation{X: -10, Y: 0}, float32(math.Pi)},
	}

	for _, tc := range cases {
		got := OrientationTo(from, tc.to)
		if math.Abs(float64(got-tc.want)) > 0.001 {
			t.Fatalf("%s: expected %.3f, got %.3f", tc.name, tc.want, got)
		}
	}
}
