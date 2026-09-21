package mapmanager

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Grid coordinate computation — mirrors AzerothCore's Acore::ComputeGridCoord
// and Acore::ComputeCellCoord from GridDefines.h
// ---------------------------------------------------------------------------

func TestComputeGridCoord_Origin(t *testing.T) {
	// WoW maps are centered: (0,0) world → center grid (32,32)
	gx, gy := ComputeGridCoord(0, 0)
	assert.Equal(t, uint32(CenterGridID), gx)
	assert.Equal(t, uint32(CenterGridID), gy)
}

func TestComputeGridCoord_PositiveXY(t *testing.T) {
	// Moving +533 yards (one grid) should shift one grid outward
	gx, gy := ComputeGridCoord(SizeOfGrids, SizeOfGrids)
	// CenterGridID - 1 = 31 (grid indices decrease with positive coords)
	assert.Equal(t, uint32(CenterGridID-1), gx)
	assert.Equal(t, uint32(CenterGridID-1), gy)
}

func TestComputeGridCoord_NegativeXY(t *testing.T) {
	gx, gy := ComputeGridCoord(-SizeOfGrids, -SizeOfGrids)
	assert.Equal(t, uint32(CenterGridID+1), gx)
	assert.Equal(t, uint32(CenterGridID+1), gy)
}

func TestComputeGridCoord_FarCorner(t *testing.T) {
	// +MAP_HALFSIZE = edge of map → grid 0
	gx, gy := ComputeGridCoord(MapHalftime, MapHalftime)
	assert.Equal(t, uint32(0), gx)
	assert.Equal(t, uint32(0), gy)

	// -MAP_HALFSIZE = opposite edge → grid 63
	gx, gy = ComputeGridCoord(-MapHalftime, -MapHalftime)
	assert.Equal(t, uint32(MaxNumberOfGrids-1), gx)
	assert.Equal(t, uint32(MaxNumberOfGrids-1), gy)
}

func TestComputeGridCoord_ClampsToValid(t *testing.T) {
	// Beyond map edge should clamp, not panic
	gx, gy := ComputeGridCoord(MapHalftime+1000, MapHalftime+1000)
	assert.Less(t, gx, uint32(MaxNumberOfGrids))
	assert.Less(t, gy, uint32(MaxNumberOfGrids))
}

// ---------------------------------------------------------------------------
// Cell coordinate computation — mirrors Acore::ComputeCellCoord
// ---------------------------------------------------------------------------

func TestComputeCellCoord_Origin(t *testing.T) {
	cx, cy := ComputeCellCoord(0, 0)
	// Center of all cells
	assert.Equal(t, uint32(CenterGridCellID), cx)
	assert.Equal(t, uint32(CenterGridCellID), cy)
}

func TestComputeCellCoord_OneCellStep(t *testing.T) {
	// Moving one cell (66.666 yards) should shift one cell
	cx, cy := ComputeCellCoord(SizeOfGridCell, SizeOfGridCell)
	assert.Equal(t, uint32(CenterGridCellID-1), cx)
	assert.Equal(t, uint32(CenterGridCellID-1), cy)
}

func TestComputeCellCoord_CrossesGridBoundary(t *testing.T) {
	// Moving SIZE_OF_GRIDS (533.333) = exactly 8 cells = one grid
	cx, cy := ComputeCellCoord(SizeOfGrids, SizeOfGrids)
	// Should be 8 cells away from center
	assert.Equal(t, uint32(CenterGridCellID-8), cx)
	assert.Equal(t, uint32(CenterGridCellID-8), cy)
}

// ---------------------------------------------------------------------------
// Cell ↔ Grid conversion
// ---------------------------------------------------------------------------

func TestCellFromCoords(t *testing.T) {
	// Cell at global coord (33, 33) → grid (4,4), cell (1,1)
	// because 33 / 8 = 4 remainder 1
	c := CellFromCoords(33, 33)
	assert.Equal(t, uint32(4), c.GridX)
	assert.Equal(t, uint32(4), c.GridY)
	assert.Equal(t, uint32(1), c.CellX)
	assert.Equal(t, uint32(1), c.CellY)
}

func TestCellFromCoords_ExactlyOnGridBoundary(t *testing.T) {
	// Global coord (8, 0) → grid (1,0), cell (0,0)
	c := CellFromCoords(8, 0)
	assert.Equal(t, uint32(1), c.GridX)
	assert.Equal(t, uint32(0), c.GridY)
	assert.Equal(t, uint32(0), c.CellX)
	assert.Equal(t, uint32(0), c.CellY)
}

func TestCellFromCoords_Zero(t *testing.T) {
	c := CellFromCoords(0, 0)
	assert.Equal(t, uint32(0), c.GridX)
	assert.Equal(t, uint32(0), c.GridY)
	assert.Equal(t, uint32(0), c.CellX)
	assert.Equal(t, uint32(0), c.CellY)
}

// ---------------------------------------------------------------------------
// Cell.CalculateCellArea — radius → bounding box of cells
// ---------------------------------------------------------------------------

func TestCalculateCellArea_ZeroRadius(t *testing.T) {
	area := CalculateCellArea(0, 0, 0)
	// Zero radius → single cell
	assert.Equal(t, area.Low, area.High)
}

func TestCalculateCellArea_SmallRadius(t *testing.T) {
	// Radius 30 yards < one cell (66.666), should span at most 2 cells
	area := CalculateCellArea(0, 0, 30)
	width := int(area.High.X) - int(area.Low.X)
	height := int(area.High.Y) - int(area.Low.Y)
	assert.LessOrEqual(t, width, 1)
	assert.LessOrEqual(t, height, 1)
}

func TestCalculateCellArea_LargeRadius(t *testing.T) {
	// Radius 200 yards ≈ 3 cells each direction
	area := CalculateCellArea(0, 0, 200)
	width := int(area.High.X) - int(area.Low.X)
	height := int(area.High.Y) - int(area.Low.Y)
	require.GreaterOrEqual(t, width, 2)
	require.GreaterOrEqual(t, height, 2)
}

func TestCalculateCellArea_VisibilityRadius(t *testing.T) {
	// Default visibility 100 yards → ~1.5 cells each way → 3×3 cells
	area := CalculateCellArea(0, 0, float64(DefaultVisibilityDistance))
	width := int(area.High.X) - int(area.Low.X)
	// 100 yards / 66.666 = 1.5 cells → should span 3 cells (±1 + center)
	assert.GreaterOrEqual(t, width, 2)
	assert.LessOrEqual(t, width, 4)
}

func TestCalculateCellArea_CenteredAtOffset(t *testing.T) {
	// Area near edge of a cell should still include neighboring cells
	area := CalculateCellArea(SizeOfGridCell*0.9, 0, 50)
	// At 0.9 cells from origin, radius 50 yards = ~0.75 cells → should span into next cell
	width := int(area.High.X) - int(area.Low.X)
	assert.GreaterOrEqual(t, width, 1)
}

// ---------------------------------------------------------------------------
// IsValidMapCoord
// ---------------------------------------------------------------------------

func TestIsValidMapCoord(t *testing.T) {
	assert.True(t, IsValidMapCoord(0, 0))
	assert.True(t, IsValidMapCoord(1000, -500))
	assert.True(t, IsValidMapCoord(MapHalftime-0.5, -(MapHalftime-0.5)))
	assert.False(t, IsValidMapCoord(MapHalftime+1, 0))
	assert.False(t, IsValidMapCoord(0, -(MapHalftime+1)))
	assert.False(t, IsValidMapCoord(math.NaN(), 0))
	assert.False(t, IsValidMapCoord(math.Inf(1), 0))
}

// ---------------------------------------------------------------------------
// Constants sanity checks
// ---------------------------------------------------------------------------

func TestGridConstants(t *testing.T) {
	assert.Equal(t, uint32(64), MaxNumberOfGrids)
	assert.Equal(t, uint32(8), MaxNumberOfCells)
	assert.InDelta(t, 533.3333, SizeOfGrids, 0.01)
	assert.InDelta(t, 66.6666, SizeOfGridCell, 0.01)
	assert.Equal(t, uint32(512), TotalCellsPerMap)
	assert.Equal(t, uint32(32), CenterGridID)
	assert.Equal(t, uint32(256), CenterGridCellID)
}
