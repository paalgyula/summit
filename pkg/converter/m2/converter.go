package m2

import (
	"fmt"
	"io"

	"github.com/paalgyula/summit/pkg/converter/gltf"
)

// ConvertToGLTF converts an M2 model (and optional .skin file) into a glTF Document.
func ConvertToGLTF(model *Model, skin *Skin, options ...Option) (*gltf.Document, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}
	var opts ConvertOptions
	for _, o := range options {
		o(&opts)
	}

	doc := gltf.NewDocument()

	// Convert vertices
	positions := make([][3]float32, len(model.Vertices))
	normals := make([][3]float32, len(model.Vertices))
	uvs := make([][2]float32, len(model.Vertices))
	joints := make([][4]uint16, len(model.Vertices))
	weights := make([][4]float32, len(model.Vertices))

	for i, v := range model.Vertices {
		positions[i] = gltf.ConvertM2ToGLTPosition(v.Pos[0], v.Pos[1], v.Pos[2])
		normals[i] = gltf.ConvertM2ToGLTPosition(v.Normal[0], v.Normal[1], v.Normal[2])
		uvs[i] = [2]float32{v.TexCoords[0], v.TexCoords[1]}

		joints[i] = [4]uint16{
			uint16(v.BoneIndices[0]),
			uint16(v.BoneIndices[1]),
			uint16(v.BoneIndices[2]),
			uint16(v.BoneIndices[3]),
		}

		totalW := float32(v.BoneWeights[0]) + float32(v.BoneWeights[1]) + float32(v.BoneWeights[2]) + float32(v.BoneWeights[3])
		if totalW > 0 {
			weights[i] = [4]float32{
				float32(v.BoneWeights[0]) / totalW,
				float32(v.BoneWeights[1]) / totalW,
				float32(v.BoneWeights[2]) / totalW,
				float32(v.BoneWeights[3]) / totalW,
			}
		} else {
			weights[i] = [4]float32{1.0, 0, 0, 0}
		}
	}

	posAcc := doc.AddFloat32Vec3Accessor(positions, gltf.TargetArrayBuffer)
	normAcc := doc.AddFloat32Vec3Accessor(normals, gltf.TargetArrayBuffer)
	uvAcc := doc.AddFloat32Vec2Accessor(uvs, gltf.TargetArrayBuffer)
	jAcc, wAcc := doc.AddJointsWeightsAccessor(joints, weights)

	attrMap := map[string]int{
		"POSITION":   posAcc,
		"NORMAL":     normAcc,
		"TEXCOORD_0": uvAcc,
	}

	if len(model.Bones) > 0 {
		attrMap["JOINTS_0"] = jAcc
		attrMap["WEIGHTS_0"] = wAcc
	}

	var primitives []gltf.Primitive

	if skin != nil && len(skin.Submeshes) > 0 {
		for subIdx, sub := range skin.Submeshes {
			var subIndices []uint16
			// Level carries the high bits of TriangleStart for large models
			triStart := int(sub.TriangleStart) + int(sub.Level)<<16
			triCount := int(sub.TriangleCount)

			if triStart+triCount <= len(skin.Triangles) {
				for t := 0; t < triCount; t++ {
					// triangles index the skin's global vertex lookup table,
					// which in turn maps to M2 vertices
					lookupIdx := int(skin.Triangles[triStart+t])
					if lookupIdx < len(skin.Indices) {
						subIndices = append(subIndices, skin.Indices[lookupIdx])
					}
				}
			}

			if len(subIndices) == 0 {
				continue
			}

			idxAcc := doc.AddUint16IndicesAccessor(subIndices)
			matIdx := len(doc.Materials)
			doc.Materials = append(doc.Materials, gltf.Material{
				Name: fmt.Sprintf("Submesh_%d_Geoset_%d", subIdx, sub.ID),
				PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
					BaseColorFactor: [4]float32{0.8, 0.8, 0.8, 1.0},
					MetallicFactor:  0.0,
					RoughnessFactor: 0.8,
				},
				DoubleSided: true,
			})

			primitives = append(primitives, gltf.Primitive{
				Attributes: attrMap,
				Indices:    &idxAcc,
				Material:   &matIdx,
			})
		}
	}

	// Fallback if no submeshes or no skin file: sequential triangle index generator
	if len(primitives) == 0 && len(positions) >= 3 {
		triCount := (len(positions) / 3) * 3
		seqIndices := make([]uint16, triCount)
		for i := 0; i < triCount; i++ {
			seqIndices[i] = uint16(i)
		}
		idxAcc := doc.AddUint16IndicesAccessor(seqIndices)
		matIdx := len(doc.Materials)
		doc.Materials = append(doc.Materials, gltf.Material{
			Name: "DefaultMaterial",
			PbrMetallicRoughness: &gltf.PbrMetallicRoughness{
				BaseColorFactor: [4]float32{0.8, 0.8, 0.8, 1.0},
				MetallicFactor:  0.0,
				RoughnessFactor: 0.8,
			},
			DoubleSided: true,
		})
		primitives = append(primitives, gltf.Primitive{
			Attributes: attrMap,
			Indices:    &idxAcc,
			Material:   &matIdx,
		})
	}

	meshIdx := len(doc.Meshes)
	meshName := model.Name
	if meshName == "" {
		meshName = "WoW_Model"
	}
	doc.Meshes = append(doc.Meshes, gltf.Mesh{
		Name:       meshName,
		Primitives: primitives,
	})

	// Add joint nodes for bones. M2 vertices are already in model space and
	// bone pivots are absolute, so each node's translation is relative to its
	// parent pivot and the inverse bind matrix undoes the absolute pivot; the
	// composed bind pose is then the identity.
	jointNodeIndices := make([]int, len(model.Bones))
	inverseBind := make([][16]float32, len(model.Bones))
	restTranslation := make([][3]float32, len(model.Bones))
	var rootBones []int
	for bIdx, bone := range model.Bones {
		jNodeIdx := len(doc.Nodes)
		jointNodeIndices[bIdx] = jNodeIdx
		pivot := gltf.ConvertM2ToGLTPosition(bone.Pivot[0], bone.Pivot[1], bone.Pivot[2])

		translation := pivot
		if bone.ParentBone >= 0 && int(bone.ParentBone) < len(model.Bones) {
			pp := model.Bones[bone.ParentBone].Pivot
			parentPivot := gltf.ConvertM2ToGLTPosition(pp[0], pp[1], pp[2])
			for i := 0; i < 3; i++ {
				translation[i] -= parentPivot[i]
			}
		} else {
			rootBones = append(rootBones, jNodeIdx)
		}

		restTranslation[bIdx] = translation

		// Column-major translate(-pivot)
		inverseBind[bIdx] = [16]float32{
			1, 0, 0, 0,
			0, 1, 0, 0,
			0, 0, 1, 0,
			-pivot[0], -pivot[1], -pivot[2], 1,
		}

		doc.Nodes = append(doc.Nodes, gltf.Node{
			Name:        fmt.Sprintf("Bone_%d_Key_%d", bIdx, bone.KeyBoneID),
			Translation: &translation,
		})
	}

	// Link bone parent-child hierarchy
	for bIdx, bone := range model.Bones {
		if bone.ParentBone >= 0 && int(bone.ParentBone) < len(jointNodeIndices) {
			parentIdx := jointNodeIndices[bone.ParentBone]
			doc.Nodes[parentIdx].Children = append(doc.Nodes[parentIdx].Children, jointNodeIndices[bIdx])
		}
	}

	var skinIdx *int
	if len(jointNodeIndices) > 0 {
		sIdx := len(doc.Skins)
		skinIdx = &sIdx
		ibmAcc := doc.AddInverseBindMatrices(inverseBind)
		skin := gltf.Skin{
			Name:                "Armature",
			Joints:              jointNodeIndices,
			InverseBindMatrices: &ibmAcc,
		}
		if len(rootBones) > 0 {
			skin.Skeleton = &rootBones[0]
		}
		doc.Skins = append(doc.Skins, skin)
	}

	// Root mesh node. The skeleton roots hang off it so the joints are part of
	// the scene graph and inherit the node's transform; otherwise loaders leave
	// the bones at the world origin and the skinned mesh is drawn there.
	rootNodeIdx := len(doc.Nodes)
	doc.Nodes = append(doc.Nodes, gltf.Node{
		Name:     meshName + "_Root",
		Mesh:     &meshIdx,
		Skin:     skinIdx,
		Children: rootBones,
	})

	doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, rootNodeIdx)

	if len(model.Bones) > 0 {
		addAnimations(doc, model, jointNodeIndices, restTranslation, opts)
	}

	return doc, nil
}

// ConvertToGLB converts an M2 model and skin directly to a GLB file stream.
func ConvertToGLB(model *Model, skin *Skin, w io.Writer, options ...Option) error {
	doc, err := ConvertToGLTF(model, skin, options...)
	if err != nil {
		return err
	}
	return doc.ToGLB(w)
}
