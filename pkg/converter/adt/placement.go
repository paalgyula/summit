package adt

import (
	"fmt"
	"strings"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// MapOrigin is the world coordinate of the map's corner: MDDF/MODF positions
// are measured from it (32 tiles in from the WoW origin on both axes).
const MapOrigin = 32 * TileSize

// PlacementExtras is attached to every doodad/WMO placeholder node in a
// terrain tile GLB. The node carries the final transform (in glTF axes); the
// client loads Model and parents it to the node.
type PlacementExtras struct {
	// Kind is "m2" for doodads, "wmo" for map objects.
	Kind string `json:"kind"`
	// Model is the asset-server path of the converted model.
	Model string `json:"model"`
	// UniqueID identifies the placement across tiles: an object near a tile
	// border is listed in every tile it overlaps, so the client draws it once.
	UniqueID uint32 `json:"uniqueId"`
	// DoodadSet selects, for WMOs, which of the building's doodad sets is
	// shown in addition to set 0.
	DoodadSet uint16 `json:"doodadSet,omitempty"`
	// NameSet is the WMO's name set (MODF.nameSet).
	NameSet uint16 `json:"nameSet,omitempty"`
	// Flags are the raw MDDF/MODF flags.
	Flags uint16 `json:"flags,omitempty"`
}

// PlacementTransform converts an MDDF/MODF placement into a glTF node
// translation, rotation quaternion and uniform scale.
//
// Placement positions are in the tile file's Y-up frame with the map corner at
// the origin; the world position is (MapOrigin - z, MapOrigin - x, y). The
// rotation is the sequence the client applies to a model-space vertex:
// Rx(rot.z - 90°), Rz(-rot.x), Ry(rot.y - 270°) in the file frame, which in
// WoW axes becomes Rz(rot.y - 270°) · Rx(-rot.x) · Ry(rot.z - 90°) · L with
// L the file-to-world axis permutation (x, y, z) -> (z, x, y).
func PlacementTransform(pos, rot [3]float32, scale float32) (translation [3]float32, rotation [4]float32, size [3]float32) {
	translation = gltf.ConvertWoWToGLTPosition(MapOrigin-pos[2], MapOrigin-pos[0], pos[1])

	fileToWorld := gltf.Mat3{{0, 0, 1}, {1, 0, 0}, {0, 1, 0}}
	r := gltf.RotationZ(gltf.Degrees(float64(rot[1]) - 270)).
		Mul(gltf.RotationX(gltf.Degrees(-float64(rot[0])))).
		Mul(gltf.RotationY(gltf.Degrees(float64(rot[2]) - 90))).
		Mul(fileToWorld)
	rotation = gltf.ConvertWoWRotation(r).Quaternion()

	size = [3]float32{scale, scale, scale}
	return translation, rotation, size
}

// addPlacementNodes appends one empty node per doodad and WMO placement to
// the document's scene.
func (adt *ADT) addPlacementNodes(doc *gltf.Document) {
	for _, d := range adt.DoodadDefs {
		if int(d.NameID) >= len(adt.ModelNames) {
			continue
		}
		t, r, s := PlacementTransform(d.Pos, d.Rot, float32(d.Scale)/1024)
		adt.appendPlacement(doc, fmt.Sprintf("doodad_%d", d.UniqueID), t, r, s, PlacementExtras{
			Kind:     "m2",
			Model:    ModelAssetPath(adt.ModelNames[d.NameID]),
			UniqueID: d.UniqueID,
			Flags:    d.Flags,
		})
	}

	for _, w := range adt.WMODefs {
		if int(w.NameID) >= len(adt.WMONames) {
			continue
		}
		t, r, s := PlacementTransform(w.Pos, w.Rot, 1)
		adt.appendPlacement(doc, fmt.Sprintf("wmo_%d", w.UniqueID), t, r, s, PlacementExtras{
			Kind:      "wmo",
			Model:     ModelAssetPath(adt.WMONames[w.NameID]),
			UniqueID:  w.UniqueID,
			DoodadSet: w.DoodadSet,
			NameSet:   w.NameSet,
			Flags:     w.Flags,
		})
	}
}

func (adt *ADT) appendPlacement(doc *gltf.Document, name string, t [3]float32, r [4]float32, s [3]float32, extras PlacementExtras) {
	nodeIdx := len(doc.Nodes)
	node := gltf.Node{
		Name:        name,
		Translation: &t,
		Rotation:    &r,
		Extras:      extras,
	}
	if s != [3]float32{1, 1, 1} {
		node.Scale = &s
	}
	doc.Nodes = append(doc.Nodes, node)
	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIdx)
}

// ModelAssetPath turns an MMDX/MWMO entry ("World\\Kalimdor\\...\\OrcTent01.mdx",
// "World\\wmo\\...\\Orgrimmar.wmo") into the asset-server path of its GLB.
func ModelAssetPath(name string) string {
	p := strings.ReplaceAll(name, "\\", "/")
	lower := strings.ToLower(p)
	for _, ext := range []string{".mdx", ".mdl", ".m2", ".wmo"} {
		if strings.HasSuffix(lower, ext) {
			p = p[:len(p)-len(ext)]
			break
		}
	}
	return p + ".glb"
}
