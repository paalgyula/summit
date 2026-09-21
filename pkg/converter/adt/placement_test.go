package adt

import (
	"math"
	"testing"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// rotate applies a glTF-space quaternion to a vector.
func rotate(q [4]float32, v [3]float64) [3]float64 {
	x, y, z, w := float64(q[0]), float64(q[1]), float64(q[2]), float64(q[3])
	// v' = v + 2w(q x v) + 2(q x (q x v))
	cx := y*v[2] - z*v[1]
	cy := z*v[0] - x*v[2]
	cz := x*v[1] - y*v[0]
	c2x := y*cz - z*cy
	c2y := z*cx - x*cz
	c2z := x*cy - y*cx
	return [3]float64{v[0] + 2*w*cx + 2*c2x, v[1] + 2*w*cy + 2*c2y, v[2] + 2*w*cz + 2*c2z}
}

func near(a, b [3]float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-4 {
			return false
		}
	}
	return true
}

func TestPlacementTransformPosition(t *testing.T) {
	// Durotar palm from Kalimdor_39_33: file (20980.9, 20.4, 17628.7) -> WoW (-562.0, -3914.3, 20.4)
	tr, _, s := PlacementTransform([3]float32{20980.9, 20.4, 17628.7}, [3]float32{}, 1134.0/1024)
	want := gltf.ConvertWoWToGLTPosition(MapOrigin-17628.7, MapOrigin-20980.9, 20.4)
	for i := range tr {
		if math.Abs(float64(tr[i]-want[i])) > 1e-2 {
			t.Fatalf("translation %v, want %v", tr, want)
		}
	}
	if math.Abs(float64(s[0]-1134.0/1024)) > 1e-6 {
		t.Fatalf("scale %v", s)
	}
}

func TestPlacementTransformRotation(t *testing.T) {
	// glTF-space model axes (ConvertM2ToGLTPosition): forward +X -> -Z, up +Z -> +Y
	up := [3]float64{0, 1, 0}
	forward := [3]float64{0, 0, -1}

	// A plain yaw keeps the model upright and turns it about the vertical axis
	_, q, _ := PlacementTransform([3]float32{}, [3]float32{0, 0, 0}, 1)
	if got := rotate(q, up); !near(got, up) {
		t.Fatalf("rot (0,0,0): up became %v", got)
	}
	if got := rotate(q, forward); !near(got, [3]float64{0, 0, 1}) {
		t.Fatalf("rot (0,0,0): forward became %v, want +Z (WoW -X)", got)
	}

	// In WoW axes the placement is Rz(rotation.y + 180°): yaw 90 -> Rz(-90°),
	// which turns WoW +X (forward) to WoW -Y, i.e. glTF +X
	_, q, _ = PlacementTransform([3]float32{}, [3]float32{0, 90, 0}, 1)
	if got := rotate(q, up); !near(got, up) {
		t.Fatalf("rot (0,90,0): up became %v", got)
	}
	if got := rotate(q, forward); !near(got, [3]float64{1, 0, 0}) {
		t.Fatalf("rot (0,90,0): forward became %v, want +X (WoW -Y)", got)
	}

	// The quaternion is unit length
	n := 0.0
	for _, c := range q {
		n += float64(c * c)
	}
	if math.Abs(n-1) > 1e-5 {
		t.Fatalf("quaternion not normalised: %v", q)
	}
}

func TestModelAssetPath(t *testing.T) {
	cases := map[string]string{
		`World\Kalimdor\Durotar\PassiveDoodads\Trees\DurotarPalm02.mdx`: "World/Kalimdor/Durotar/PassiveDoodads/Trees/DurotarPalm02.glb",
		`world\wmo\kalimdor\buildings\orctwostory\orctwostory.wmo`:      "world/wmo/kalimdor/buildings/orctwostory/orctwostory.glb",
		`Creature\Boar\Boar.m2`: "Creature/Boar/Boar.glb",
	}
	for in, want := range cases {
		if got := ModelAssetPath(in); got != want {
			t.Errorf("ModelAssetPath(%q) = %q, want %q", in, got, want)
		}
	}
}
