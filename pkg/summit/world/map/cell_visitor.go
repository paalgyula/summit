package mapmanager

import (
	"github.com/paalgyula/summit/pkg/summit/world/object"
)

// ---------------------------------------------------------------------------
// GridCell — stores objects within a single cell (66.666 × 66.666 yards)
// ---------------------------------------------------------------------------

// GridCell holds all objects located within a single cell.
// Objects are split into two containers: normal and far-visible.
type GridCell struct {
	objects     []*object.Object
	farVisible  []*object.Object
}

// NewGridCell creates an empty GridCell.
func NewGridCell() *GridCell {
	return &GridCell{}
}

// ObjectCount returns total number of objects in the cell.
func (c *GridCell) ObjectCount() int {
	return len(c.objects)
}

// FarVisibleCount returns the number of far-visible objects.
func (c *GridCell) FarVisibleCount() int {
	return len(c.farVisible)
}

// AddObject adds an object to the cell.
func (c *GridCell) AddObject(obj *object.Object) {
	c.objects = append(c.objects, obj)
	if obj.IsFarVisible() {
		c.farVisible = append(c.farVisible, obj)
	}
}

// RemoveObject removes an object from the cell.
func (c *GridCell) RemoveObject(obj *object.Object) {
	for i, o := range c.objects {
		if o == obj {
			c.objects = append(c.objects[:i], c.objects[i+1:]...)
			break
		}
	}
	if obj.IsFarVisible() {
		for i, o := range c.farVisible {
			if o == obj {
				c.farVisible = append(c.farVisible[:i], c.farVisible[i+1:]...)
				break
			}
		}
	}
}

// Visit iterates all objects in the cell. Return false to stop early.
// Returns true if iteration completed, false if stopped early.
func (c *GridCell) Visit(fn func(obj *object.Object) bool) bool {
	for _, obj := range c.objects {
		if !fn(obj) {
			return false
		}
	}
	return true
}

// VisitFarVisible iterates only far-visible objects. Return false to stop early.
// Returns true if iteration completed, false if stopped early.
func (c *GridCell) VisitFarVisible(fn func(obj *object.Object) bool) bool {
	for _, obj := range c.farVisible {
		if !fn(obj) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// MapGrid — 8×8 cells within a single grid (533.333 × 533.333 yards)
// ---------------------------------------------------------------------------

// MapGrid holds 8×8 cells for one grid on the map.
type MapGrid struct {
	gridX, gridY uint32
	cells        [MaxNumberOfCells][MaxNumberOfCells]*GridCell
	objectCount  int
}

// NewMapGrid creates a new MapGrid at the given grid coordinates.
func NewMapGrid(gx, gy uint32) *MapGrid {
	return &MapGrid{gridX: gx, gridY: gy}
}

// getOrCreateCell returns the cell, creating it if needed.
func (g *MapGrid) getOrCreateCell(cx, cy uint32) *GridCell {
	if g.cells[cx][cy] == nil {
		g.cells[cx][cy] = NewGridCell()
	}
	return g.cells[cx][cy]
}

// AddObject adds an object to the specified cell within this grid.
func (g *MapGrid) AddObject(cx, cy uint32, obj *object.Object) {
	g.getOrCreateCell(cx, cy).AddObject(obj)
	g.objectCount++
}

// RemoveObject removes an object from the specified cell.
func (g *MapGrid) RemoveObject(cx, cy uint32, obj *object.Object) {
	if cell := g.cells[cx][cy]; cell != nil {
		cell.RemoveObject(obj)
		g.objectCount--
	}
}

// ObjectCount returns the total number of objects across all cells.
func (g *MapGrid) ObjectCount() int {
	return g.objectCount
}

// VisitCell iterates objects in a specific cell.
// Returns true if iteration completed, false if stopped early.
func (g *MapGrid) VisitCell(cx, cy uint32, fn func(obj *object.Object) bool) bool {
	if cell := g.cells[cx][cy]; cell != nil {
		return cell.Visit(fn)
	}
	return true
}

// VisitAllCells iterates all objects in all cells of this grid.
// Returns true if iteration completed, false if stopped early.
func (g *MapGrid) VisitAllCells(fn func(obj *object.Object) bool) bool {
	for cx := uint32(0); cx < MaxNumberOfCells; cx++ {
		for cy := uint32(0); cy < MaxNumberOfCells; cy++ {
			if cell := g.cells[cx][cy]; cell != nil {
				if !cell.Visit(fn) {
					return false
				}
			}
		}
	}
	return true
}

// VisitAllFarVisible iterates only far-visible objects across all cells.
// Returns true if iteration completed, false if stopped early.
func (g *MapGrid) VisitAllFarVisible(fn func(obj *object.Object) bool) bool {
	for cx := uint32(0); cx < MaxNumberOfCells; cx++ {
		for cy := uint32(0); cy < MaxNumberOfCells; cy++ {
			if cell := g.cells[cx][cy]; cell != nil {
				if !cell.VisitFarVisible(fn) {
					return false
				}
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Map spatial queries — VisitObjects, VisitFarVisibleObjects
// ---------------------------------------------------------------------------

// AddObjectToGrid adds an object to the map's grid at the given world position.
func (m *Map) AddObjectToGrid(obj *object.Object, x, y float32, z float32) {
	gx, gy := ComputeGridCoord(float64(x), float64(y))
	cx, cy := ComputeCellCoord(float64(x), float64(y))
	localCX := cx % MaxNumberOfCells
	localCY := cy % MaxNumberOfCells

	grid := m.getOrCreateGrid(gx, gy)
	grid.AddObject(localCX, localCY, obj)
}

// getOrCreateGrid returns the grid at (gx,gy), creating it if needed.
func (m *Map) getOrCreateGrid(gx, gy uint32) *MapGrid {
	if m.grids == nil {
		m.grids = make(map[uint32]*MapGrid)
	}
	key := gy*MaxNumberOfGrids + gx
	if g, ok := m.grids[key]; ok {
		return g
	}
	g := NewMapGrid(gx, gy)
	m.grids[key] = g
	return g
}

// VisitObjects calls fn for every object within radius of (x, y).
// Mirrors AzerothCore's Cell::VisitObjects.
func (m *Map) VisitObjects(x, y, radius float32, fn func(obj *object.Object) bool) {
	if radius <= 0 {
		return
	}

	// Cap radius at grid size (like AzerothCore)
	if radius > float32(SizeOfGrids) {
		radius = float32(SizeOfGrids)
	}

	area := CalculateCellArea(float64(x), float64(y), float64(radius))

	for cx := area.Low.X; cx <= area.High.X; cx++ {
		for cy := area.Low.Y; cy <= area.High.Y; cy++ {
			cellCoord := CellFromCoords(cx, cy)
			grid := m.getGrid(cellCoord.GridX, cellCoord.GridY)
			if grid == nil {
				continue
			}
			if !grid.VisitCell(cellCoord.CellX, cellCoord.CellY, fn) {
				return
			}
		}
	}
}

// VisitFarVisibleObjects calls fn for every far-visible object within radius.
// Mirrors AzerothCore's Cell::VisitFarVisibleObjects.
func (m *Map) VisitFarVisibleObjects(x, y, radius float32, fn func(obj *object.Object) bool) {
	if radius <= 0 {
		return
	}

	if radius > float32(SizeOfGrids) {
		radius = float32(SizeOfGrids)
	}

	area := CalculateCellArea(float64(x), float64(y), float64(radius))

	for cx := area.Low.X; cx <= area.High.X; cx++ {
		for cy := area.Low.Y; cy <= area.High.Y; cy++ {
			cellCoord := CellFromCoords(cx, cy)
			grid := m.getGrid(cellCoord.GridX, cellCoord.GridY)
			if grid == nil {
				continue
			}
			// Visit far-visible objects in this cell
			cell := grid.getCell(cellCoord.CellX, cellCoord.CellY)
			if cell != nil {
				if !cell.VisitFarVisible(fn) {
					return
				}
			}
		}
	}
}

// getGrid returns the grid at (gx,gy) or nil if not created yet.
func (m *Map) getGrid(gx, gy uint32) *MapGrid {
	if m.grids == nil {
		return nil
	}
	key := gy*MaxNumberOfGrids + gx
	return m.grids[key]
}

// getCell returns the cell at (cx,cy) within the grid or nil.
func (g *MapGrid) getCell(cx, cy uint32) *GridCell {
	if cx >= MaxNumberOfCells || cy >= MaxNumberOfCells {
		return nil
	}
	return g.cells[cx][cy]
}
