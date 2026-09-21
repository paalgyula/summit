package mapmanager

import "math"

// ---------------------------------------------------------------------------
// Grid system constants — ported from AzerothCore's MapDefines.h + GridDefines.h
// ---------------------------------------------------------------------------

const (
	MaxNumberOfGrids   uint32 = 64
	MaxNumberOfCells   uint32 = 8
	SizeOfGrids               = 533.3333
	SizeOfGridCell            = SizeOfGrids / float64(MaxNumberOfCells) // 66.6666
	TotalCellsPerMap  uint32 = MaxNumberOfGrids * MaxNumberOfCells     // 512
	CenterGridID      uint32 = MaxNumberOfGrids / 2                    // 32
	CenterGridCellID  uint32 = TotalCellsPerMap / 2                    // 256
	MapSize                  = SizeOfGrids * float64(MaxNumberOfGrids) // 34133.33
	MapHalftime              = MapSize / 2.0
)

// ---------------------------------------------------------------------------
// GridCoord — position in grid space (0..63 per axis)
// ---------------------------------------------------------------------------

type GridCoord struct {
	X, Y uint32
}

// ---------------------------------------------------------------------------
// CellCoord — position in cell space (0..511 per axis)
// ---------------------------------------------------------------------------

type CellCoord struct {
	X, Y uint32
}

// ---------------------------------------------------------------------------
// Cell — grid_x,grid_y + cell_x,cell_y breakdown
// ---------------------------------------------------------------------------

type Cell struct {
	GridX, GridY uint32
	CellX, CellY uint32
}

// CellArea is the bounding box of cells for a radius query.
type CellArea struct {
	Low, High CellCoord
}

// CellFromCoords converts global cell coordinates to a Cell (grid + cell indices).
func CellFromCoords(cx, cy uint32) Cell {
	return Cell{
		GridX: cx / MaxNumberOfCells,
		GridY: cy / MaxNumberOfCells,
		CellX: cx % MaxNumberOfCells,
		CellY: cy % MaxNumberOfCells,
	}
}

// ---------------------------------------------------------------------------
// Coordinate computation — mirrors AzerothCore's Acore::Compute* functions
// ---------------------------------------------------------------------------

// ComputeGridCoord converts world (x,y) to grid coordinates.
// Mirrors: Acore::ComputeGridCoord in GridDefines.h
func ComputeGridCoord(x, y float64) (uint32, uint32) {
	gx := int(float64(CenterGridID) - x/SizeOfGrids)
	gy := int(float64(CenterGridID) - y/SizeOfGrids)

	// Clamp to valid range [0, MAX_NUMBER_OF_GRIDS-1]
	if gx < 0 {
		gx = 0
	}
	if gy < 0 {
		gy = 0
	}
	if gx >= int(MaxNumberOfGrids) {
		gx = int(MaxNumberOfGrids) - 1
	}
	if gy >= int(MaxNumberOfGrids) {
		gy = int(MaxNumberOfGrids) - 1
	}

	return uint32(gx), uint32(gy)
}

// ComputeCellCoord converts world (x,y) to cell coordinates.
// Mirrors: Acore::ComputeCellCoord in GridDefines.h
func ComputeCellCoord(x, y float64) (uint32, uint32) {
	cx := int(float64(CenterGridCellID) - x/SizeOfGridCell)
	cy := int(float64(CenterGridCellID) - y/SizeOfGridCell)

	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	if cx >= int(TotalCellsPerMap) {
		cx = int(TotalCellsPerMap) - 1
	}
	if cy >= int(TotalCellsPerMap) {
		cy = int(TotalCellsPerMap) - 1
	}

	return uint32(cx), uint32(cy)
}

// CalculateCellArea returns the bounding box of cells that overlap with a circle
// of the given radius centered at (x, y).
// Mirrors: Cell::CalculateCellArea in CellImpl.h
func CalculateCellArea(x, y, radius float64) CellArea {
	if radius <= 0 {
		cx, cy := ComputeCellCoord(x, y)
		return CellArea{Low: CellCoord{cx, cy}, High: CellCoord{cx, cy}}
	}

	hx, hy := ComputeCellCoord(x+radius, y+radius)
	lx, ly := ComputeCellCoord(x-radius, y-radius)

	// Normalize: ensure low ≤ high
	if lx > hx {
		lx, hx = hx, lx
	}
	if ly > hy {
		ly, hy = hy, ly
	}

	return CellArea{
		Low:  CellCoord{lx, ly},
		High: CellCoord{hx, hy},
	}
}

// IsValidMapCoord returns true if (x, y) is within the map boundaries.
func IsValidMapCoord(x, y float64) bool {
	return !math.IsInf(x, 0) && !math.IsNaN(x) &&
		!math.IsInf(y, 0) && !math.IsNaN(y) &&
		math.Abs(x) <= MapHalftime-0.5 &&
		math.Abs(y) <= MapHalftime-0.5
}
