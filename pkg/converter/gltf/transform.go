package gltf

import "math"

// Mat3 is a row-major 3x3 matrix acting on column vectors (v' = M v).
type Mat3 [3][3]float64

// Identity returns the identity matrix.
func Identity() Mat3 {
	return Mat3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

// Mul returns a·b, i.e. the transform that applies b first, then a.
func (a Mat3) Mul(b Mat3) Mat3 {
	var out Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			out[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j] + a[i][2]*b[2][j]
		}
	}
	return out
}

// Transpose returns the transposed matrix (the inverse for rotations).
func (a Mat3) Transpose() Mat3 {
	var out Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			out[i][j] = a[j][i]
		}
	}
	return out
}

// Apply returns a·v.
func (a Mat3) Apply(v [3]float64) [3]float64 {
	return [3]float64{
		a[0][0]*v[0] + a[0][1]*v[1] + a[0][2]*v[2],
		a[1][0]*v[0] + a[1][1]*v[1] + a[1][2]*v[2],
		a[2][0]*v[0] + a[2][1]*v[1] + a[2][2]*v[2],
	}
}

// RotationX is a right-handed rotation about +X by rad.
func RotationX(rad float64) Mat3 {
	c, s := math.Cos(rad), math.Sin(rad)
	return Mat3{{1, 0, 0}, {0, c, -s}, {0, s, c}}
}

// RotationY is a right-handed rotation about +Y by rad.
func RotationY(rad float64) Mat3 {
	c, s := math.Cos(rad), math.Sin(rad)
	return Mat3{{c, 0, s}, {0, 1, 0}, {-s, 0, c}}
}

// RotationZ is a right-handed rotation about +Z by rad.
func RotationZ(rad float64) Mat3 {
	c, s := math.Cos(rad), math.Sin(rad)
	return Mat3{{c, -s, 0}, {s, c, 0}, {0, 0, 1}}
}

// Quaternion converts a rotation matrix into a unit quaternion (x, y, z, w),
// the glTF node rotation layout.
func (a Mat3) Quaternion() [4]float32 {
	trace := a[0][0] + a[1][1] + a[2][2]
	var x, y, z, w float64
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1) * 2
		w = 0.25 * s
		x = (a[2][1] - a[1][2]) / s
		y = (a[0][2] - a[2][0]) / s
		z = (a[1][0] - a[0][1]) / s
	case a[0][0] > a[1][1] && a[0][0] > a[2][2]:
		s := math.Sqrt(1+a[0][0]-a[1][1]-a[2][2]) * 2
		w = (a[2][1] - a[1][2]) / s
		x = 0.25 * s
		y = (a[0][1] + a[1][0]) / s
		z = (a[0][2] + a[2][0]) / s
	case a[1][1] > a[2][2]:
		s := math.Sqrt(1+a[1][1]-a[0][0]-a[2][2]) * 2
		w = (a[0][2] - a[2][0]) / s
		x = (a[0][1] + a[1][0]) / s
		y = 0.25 * s
		z = (a[1][2] + a[2][1]) / s
	default:
		s := math.Sqrt(1+a[2][2]-a[0][0]-a[1][1]) * 2
		w = (a[1][0] - a[0][1]) / s
		x = (a[0][2] + a[2][0]) / s
		y = (a[1][2] + a[2][1]) / s
		z = 0.25 * s
	}
	return [4]float32{float32(x), float32(y), float32(z), float32(w)}
}

// wowToGLTF is ConvertWoWToGLTPosition as a matrix: (x, y, z) -> (-y, z, -x).
var wowToGLTF = Mat3{{0, -1, 0}, {0, 0, 1}, {-1, 0, 0}}

// ConvertWoWRotation re-expresses a rotation given in WoW axes in glTF axes,
// so that it acts on vertices already converted with ConvertWoWToGLTPosition.
func ConvertWoWRotation(r Mat3) Mat3 {
	return wowToGLTF.Mul(r).Mul(wowToGLTF.Transpose())
}

// ConvertWoWQuaternion re-expresses a WoW-space unit quaternion (x, y, z, w)
// in glTF axes: the axis is permuted like a position, the angle is unchanged.
func ConvertWoWQuaternion(q [4]float32) [4]float32 {
	return [4]float32{-q[1], q[2], -q[0], q[3]}
}

// Degrees converts degrees to radians.
func Degrees(deg float64) float64 {
	return deg * math.Pi / 180
}

// LookAtRotation returns the quaternion orienting a glTF camera (which looks
// down its local -Z with +Y up) at target from position.
func LookAtRotation(position, target [3]float32) [4]float32 {
	f := [3]float64{float64(target[0] - position[0]), float64(target[1] - position[1]), float64(target[2] - position[2])}
	l := math.Sqrt(f[0]*f[0] + f[1]*f[1] + f[2]*f[2])
	if l == 0 {
		return [4]float32{0, 0, 0, 1}
	}
	for i := range f {
		f[i] /= l
	}
	up := [3]float64{0, 1, 0}
	if math.Abs(f[1]) > 0.999 {
		up = [3]float64{0, 0, 1}
	}
	// camera axes: z = -forward, x = up × z, y = z × x
	z := [3]float64{-f[0], -f[1], -f[2]}
	x := cross(up, z)
	xl := math.Sqrt(x[0]*x[0] + x[1]*x[1] + x[2]*x[2])
	for i := range x {
		x[i] /= xl
	}
	y := cross(z, x)
	// columns are the camera axes
	m := Mat3{{x[0], y[0], z[0]}, {x[1], y[1], z[1]}, {x[2], y[2], z[2]}}
	return m.Quaternion()
}

func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
