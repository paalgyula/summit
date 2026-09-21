package mapmanager

import (
	"testing"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GridCell — stores objects within a single cell
// ---------------------------------------------------------------------------

func TestGridCell_AddAndCount(t *testing.T) {
	cell := NewGridCell()
	assert.Equal(t, 0, cell.ObjectCount())

	obj := object.NewObject()
	cell.AddObject(obj)
	assert.Equal(t, 1, cell.ObjectCount())
}

func TestGridCell_AddMultiple(t *testing.T) {
	cell := NewGridCell()
	for i := 0; i < 5; i++ {
		cell.AddObject(object.NewObject())
	}
	assert.Equal(t, 5, cell.ObjectCount())
}

func TestGridCell_Remove(t *testing.T) {
	cell := NewGridCell()
	obj := object.NewObject()
	cell.AddObject(obj)
	assert.Equal(t, 1, cell.ObjectCount())

	cell.RemoveObject(obj)
	assert.Equal(t, 0, cell.ObjectCount())
}

func TestGridCell_FarVisible(t *testing.T) {
	cell := NewGridCell()

	normal := object.NewObject()
	far := object.NewObject()
	far.SetVisibilityOverrideType(object.VisibilityDistanceLarge)

	cell.AddObject(normal)
	cell.AddObject(far)

	assert.Equal(t, 2, cell.ObjectCount())
	assert.Equal(t, 1, cell.FarVisibleCount())
}

func TestGridCell_Visit(t *testing.T) {
	cell := NewGridCell()
	for i := 0; i < 3; i++ {
		cell.AddObject(object.NewObject())
	}

	var visited []*object.Object
	cell.Visit(func(obj *object.Object) bool {
		visited = append(visited, obj)
		return true // continue
	})
	assert.Equal(t, 3, len(visited))
}

func TestGridCell_VisitStop(t *testing.T) {
	cell := NewGridCell()
	for i := 0; i < 5; i++ {
		cell.AddObject(object.NewObject())
	}

	count := 0
	cell.Visit(func(obj *object.Object) bool {
		count++
		return count < 2 // stop after 2
	})
	assert.Equal(t, 2, count)
}

// ---------------------------------------------------------------------------
// MapGrid — 8×8 cells within a grid
// ---------------------------------------------------------------------------

func TestMapGrid_AddToObject(t *testing.T) {
	g := NewMapGrid(0, 0)
	obj := object.NewObject()

	// Add at cell (0,0) within the grid
	g.AddObject(0, 0, obj)
	assert.Equal(t, 1, g.ObjectCount())
}

func TestMapGrid_CellCount(t *testing.T) {
	g := NewMapGrid(0, 0)
	assert.Equal(t, 0, g.ObjectCount())

	g.AddObject(0, 0, object.NewObject())
	g.AddObject(0, 1, object.NewObject())
	g.AddObject(1, 0, object.NewObject())
	assert.Equal(t, 3, g.ObjectCount())
}

func TestMapGrid_VisitCell(t *testing.T) {
	g := NewMapGrid(0, 0)
	obj := object.NewObject()
	g.AddObject(3, 3, obj)

	var found []*object.Object
	g.VisitCell(3, 3, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})
	require.Len(t, found, 1)
	assert.Equal(t, obj, found[0])
}

// ---------------------------------------------------------------------------
// Map.VisitObjects — radius query across the grid
// ---------------------------------------------------------------------------

func TestMap_VisitObjects_SingleCell(t *testing.T) {
	m := NewMap(0, 0, nil)

	// Create an object at origin
	obj := object.NewObject()
	m.AddObjectToGrid(obj, 0, 0, 0)

	// Visit with radius that only covers the center cell
	var found []*object.Object
	m.VisitObjects(0, 0, 10, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	require.Len(t, found, 1)
	assert.Equal(t, obj, found[0])
}

func TestMap_VisitObjects_MultipleObjects(t *testing.T) {
	m := NewMap(0, 0, nil)

	// Add several objects near origin
	for i := 0; i < 5; i++ {
		obj := object.NewObject()
		m.AddObjectToGrid(obj, 0, 0, 0)
	}

	var found []*object.Object
	m.VisitObjects(0, 0, 10, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	assert.Equal(t, 5, len(found))
}

func TestMap_VisitObjects_OutOfRange(t *testing.T) {
	m := NewMap(0, 0, nil)

	// Add object at far corner of map
	obj := object.NewObject()
	farX := float32(SizeOfGrids * 10)
	m.AddObjectToGrid(obj, farX, 0, 0)

	// Visit near origin — should not find the far object
	var found []*object.Object
	m.VisitObjects(0, 0, 50, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	assert.Empty(t, found)
}

func TestMap_VisitObjects_RadiusCoversMultipleCells(t *testing.T) {
	m := NewMap(0, 0, nil)

	// Add objects in adjacent cells
	obj1 := object.NewObject()
	obj2 := object.NewObject()
	m.AddObjectToGrid(obj1, 0, 0, 0)
	m.AddObjectToGrid(obj2, 30, 0, 0) // 30 yards away, still within one cell

	var found []*object.Object
	m.VisitObjects(0, 0, 50, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	assert.Equal(t, 2, len(found))
}

// ---------------------------------------------------------------------------
// Map.VisitFarVisibleObjects — far-visible only
// ---------------------------------------------------------------------------

func TestMap_VisitFarVisibleObjects_OnlyFarVisible(t *testing.T) {
	m := NewMap(0, 0, nil)

	normal := object.NewObject()
	far := object.NewObject()
	far.SetVisibilityOverrideType(object.VisibilityDistanceLarge)

	m.AddObjectToGrid(normal, 0, 0, 0)
	m.AddObjectToGrid(far, 0, 0, 0)

	var found []*object.Object
	m.VisitFarVisibleObjects(0, 0, 300, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	require.Len(t, found, 1)
	assert.Equal(t, far, found[0])
}

func TestMap_VisitFarVisibleObjects_WiderRadius(t *testing.T) {
	m := NewMap(0, 0, nil)

	// Far-visible object at 200 yards (beyond normal visibility, within large)
	far := object.NewObject()
	far.SetVisibilityOverrideType(object.VisibilityDistanceLarge)
	m.AddObjectToGrid(far, 200, 0, 0)

	// Normal object at same distance — should not be found by far-visible query
	normal := object.NewObject()
	m.AddObjectToGrid(normal, 200, 0, 0)

	var found []*object.Object
	m.VisitFarVisibleObjects(0, 0, 300, func(o *object.Object) bool {
		found = append(found, o)
		return true
	})

	require.Len(t, found, 1)
	assert.Equal(t, far, found[0])
}
