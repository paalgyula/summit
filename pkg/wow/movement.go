package wow

type MoveType int

const (
	MoveTypeWalk MoveType = iota
	MoveTypeRun
	MoveTypeRunBack
	MoveTypeSwim
	MoveTypeSwimBack
	MoveTypeTurnRate
	MoveTypeFlight
	MoveTypeFlightBack
	MoveTypeMax
)

// const MoveTypeMax = MoveTypeFlightBack + 1

type MovementFlag uint32

const (
	MovementFlagNone            MovementFlag = 0x00000000
	MovementFlagForward         MovementFlag = 0x00000001
	MovementFlagBackward        MovementFlag = 0x00000002
	MovementFlagStrafeLeft      MovementFlag = 0x00000004
	MovementFlagStrafeRight     MovementFlag = 0x00000008
	MovementFlagTurnLeft        MovementFlag = 0x00000010
	MovementFlagTurnRight       MovementFlag = 0x00000020
	MovementFlagPitchUp         MovementFlag = 0x00000040
	MovementFlagPitchDown       MovementFlag = 0x00000080
	MovementFlagWalkMode        MovementFlag = 0x00000100 // Walking
	MovementFlagOnTransport     MovementFlag = 0x00000200 // Used for flying on some creatures
	MovementFlagLevitating      MovementFlag = 0x00000400
	MovementFlagRoot            MovementFlag = 0x00000800
	MovementFlagFalling         MovementFlag = 0x00001000
	MovementFlagFallingFar      MovementFlag = 0x00004000
	MovementFlagSwimming        MovementFlag = 0x00200000 // appears with fly flag also
	MovementFlagAscending       MovementFlag = 0x00400000 // swim up also
	MovementFlagCanFly          MovementFlag = 0x00800000
	MovementFlagFlying          MovementFlag = 0x01000000
	MovementFlagFlying2         MovementFlag = 0x02000000 // Actual flying mode
	MovementFlagSplineElevation MovementFlag = 0x04000000 // used for flight paths
	MovementFlagSplineEnabled   MovementFlag = 0x08000000 // used for flight paths
	MovementFlagWaterwalking    MovementFlag = 0x10000000 // prevent unit from falling through water
	MovementFlagSafeFall        MovementFlag = 0x20000000 // Feather Fall (spell)
	MovementFlagHover           MovementFlag = 0x40000000

	MovementFlagMoving MovementFlag = MovementFlagForward | MovementFlagBackward | MovementFlagStrafeLeft | MovementFlagStrafeRight |
		MovementFlagPitchUp | MovementFlagPitchDown |
		MovementFlagFalling | MovementFlagFallingFar | MovementFlagAscending |
		MovementFlagSplineElevation

	MovementFlagTurning MovementFlag = MovementFlagTurnLeft | MovementFlagTurnRight

	MovementFlagMaskMovingFly MovementFlag = MovementFlagFlying2 | MovementFlagAscending | MovementFlagCanFly
)
