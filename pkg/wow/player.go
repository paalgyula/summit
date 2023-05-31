package wow

type PlayerClass uint8

// TODO: generate this from dbc?
const (
	ClassWarior      PlayerClass = 0x01
	ClassPaladin     PlayerClass = 0x02
	ClassHunter      PlayerClass = 0x03
	ClassRogue       PlayerClass = 0x04
	ClassPriest      PlayerClass = 0x05
	ClassDeathKnight PlayerClass = 0x06
	ClassShaman      PlayerClass = 0x07
	ClassMage        PlayerClass = 0x08
	ClassWarlock     PlayerClass = 0x09
	ClassDruid       PlayerClass = 0x0b
)

type PlayerRace uint8

const (
	RaceHuman    PlayerRace = 0x01
	RaceDwarf    PlayerRace = 0x03
	RaceNightElf PlayerRace = 0x04
	RaceGnome    PlayerRace = 0x07
	RaceDraenei  PlayerRace = 0x0b
)

type PlayerGender uint8

const (
	GenderMale   PlayerGender = 0x00
	GenderFemale PlayerGender = 0x01
)

type PlayerFlag uint32

// PlayerFlag values.
const (
	PlayerFlagsGroupLeader     PlayerFlag = 0x00001
	PlayerFlagsAFK             PlayerFlag = 0x00002
	PlayerFlagsDND             PlayerFlag = 0x00004
	PlayerFlagsGM              PlayerFlag = 0x00008
	PlayerFlagsGhost           PlayerFlag = 0x00010
	PlayerFlagsResting         PlayerFlag = 0x00020
	PlayerFlagsFFAPVP          PlayerFlag = 0x00080
	PlayerFlagsContestedPVP    PlayerFlag = 0x00100
	PlayerFlagsInPVP           PlayerFlag = 0x00200
	PlayerFlagsHideHelm        PlayerFlag = 0x00400
	PlayerFlagsHideCloak       PlayerFlag = 0x00800
	PlayerFlagsPartialPlayTime PlayerFlag = 0x01000
	PlayerFlagsNoPlayTime      PlayerFlag = 0x02000
	PlayerFlagsSanctuary       PlayerFlag = 0x10000
	PlayerFlagsTaxiBenchmark   PlayerFlag = 0x20000
	PlayerFlagsPVPTimer        PlayerFlag = 0x40000
)
