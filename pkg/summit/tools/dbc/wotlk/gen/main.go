// dbc-gen generates Go DBC struct definitions from AzerothCore format strings.
// Run: go run ./pkg/summit/tools/dbc/wotlk/gen
// Output: one .go file per DBC in the parent directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// DBC definitions: name -> format string from AzerothCore DBCfmt.h
// Characters: i=uint32, f=float32, s=string(uint32 offset), x=skip(4b), X=skip(1b), b=uint8, n/d=index(uint32), l=bool
var dbcs = map[string]string{
	// Character (ChrClasses, ChrRaces, CharStartOutfit, CharTitles) handled manually
	// CharStartOutfit is handled manually in character.go (needs slice types)
	// CharTitles handled manually

	// Map / World
	"Map":              "nxiixssssssssssssssssxixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxixiffxixi",
	"MapDifficulty":    "diisxxxxxxxxxxxxxxxxiix",
	"AreaTable":        "niiiixxxxxissssssssssssssssxiiiiixxx",
	"AreaGroup":        "niiiiiii",
	"AreaPOI":          "niiiiiiiiiiifffixixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxix",
	"WMOAreaTable":     "niiixxxxxiixxxxxxxxxxxxxxxxx",
	"WorldMapArea":     "xinxffffixx",
	"WorldMapOverlay":  "nxiiiixxxxxxxxxxx",
	"Light":            "nifffxxxxxxxxxx",
	"LiquidType":       "nxxixixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",

	// Spells
	"SpellEntry":                    "niiiiiiiiiiiixixiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiifxiiiiiiiiiiiiiiiiiiiiiiiiiiiifffiiiiiiiiiiiiiiiiiiiiifffiiiiiiiiiiiiiiifffiiiiiiiiiiiiiissssssssssssssssxssssssssssssssssxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxiiiiiiiiiiixfffxxxiiiiixxfffxx",
	"SpellCastTime":                 "nixx",
	"SpellCategory":                 "ni",
	"SpellDifficulty":               "niiii",
	"SpellDuration":                 "niii",
	"SpellFocusObject":              "nxxxxxxxxxxxxxxxxx",
	"SpellRadius":                   "nfff",
	"SpellRange":                    "nffffixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	"SpellRuneCost":                 "niiii",
	"SpellItemEnchantment":          "niiiiiiixxxiiissssssssssssssssxiiiiiii",
	"SpellItemEnchantmentCondition": "nbbbbbxxxxxbbbbbbbbbbiiiiiXXXXX",
	"SpellShapeshiftForm":           "nxxxxxxxxxxxxxxxxxxiixiiixxiiiiiiii",
	"SpellVisual":                   "dxxxxxxiixxxxxxxxxxxxxxxxxxxxxxx",

	// Skills / Talents
	"SkillLine":           "nixssssssssssssssssxxxxxxxxxxxxxxxxxxixxxxxxxxxxxxxxxxxi",
	"SkillLineAbility":    "niiiixxiiiiixx",
	"SkillRaceClassInfo":  "diiiixix",
	"SkillTiers":          "nxxxxxxxxxxxxxxxxiiiiiiiiiiiiiiii",
	"Talent":              "niiiiiiiixxxxixxixxixxx",
	"TalentTab":           "nxxxxxxxxxxxxxxxxxxxiiix",

	// Items
	"Item":                  "niiiiiii",
	"ItemBagFamily":         "nxxxxxxxxxxxxxxxxx",
	"ItemDisplayInfo":       "nxxxxsxxxxxxxxxxxxxxxxxxx",
	"ItemExtendedCost":      "niiiiiiiiiiiiiix",
	"ItemLimitCategory":     "nxxxxxxxxxxxxxxxxxii",
	"ItemRandomProperties":  "nxiiiiissssssssssssssssx",
	"ItemRandomSuffix":      "nssssssssssssssssxxiiiiiiiiii",
	"ItemSet":               "dssssssssssssssssxiiiiiiiiiixxxxxxxiiiiiiiiiiiiiiiiii",

	// Creatures
	"CreatureDisplayInfo":      "nixifxxxxxxxxxxx",
	"CreatureDisplayInfoExtra": "dixxxxxxxxxxxxxxxxxxx",
	"CreatureFamily":           "nfifiiiiixssssssssssssssssxx",
	"CreatureModelData":        "nixxfxxxxxxxxxfffxxxxxxxxxxx",
	"CreatureSpellData":        "niiiixxxx",
	"CreatureType":             "nxxxxxxxxxxxxxxxxxx",

	// Game Objects / Vehicles
	"GameObjectArtKit":      "nxxxxxxx",
	"GameObjectDisplayInfo": "nsxxxxxxxxxxffffffx",
	"Vehicle":               "niffffiiiiiiiifffffffffffffffssssfifiixx",
	"VehicleSeat":           "niiffffffffffiiiiiifffffffiiifffiiiiiiiffiiiiixxxxxxxxxxxx",
	"TransportAnimation":    "diifffx",
	"TransportRotation":     "diiffff",

	// Spells - Gt* tables
	"GtBarberShopCostBase":         "df",
	"GtChanceToMeleeCrit":          "df",
	"GtChanceToMeleeCritBase":      "df",
	"GtChanceToSpellCrit":          "df",
	"Faction":                      "niiiiiiiiiiiiiiiiiiffixssssssssssssssssxxxxxxxxxxxxxxxxxx",
	"GtChanceToSpellCritBase":      "df",
	"GtCombatRatings":              "df",
	"GtNPCManaCostScaler":          "df",
	"GtOCTClassCombatRatingScalar": "df",
	"GtOCTRegenHP":                 "df",
	"GtRegenHPPerSpt":              "df",
	"GtRegenMPPerSpt":              "df",

	// Achievements
	"Achievement":         "niixssssssssssssssssxxxxxxxxxxxxxxxxxxiixixxxxxxxxxxxxxxxxxxii",
	"AchievementCategory": "nixxxxxxxxxxxxxxxxxx",
	"AchievementCriteria": "niiiiiiiixxxxxxxxxxxxxxxxxiiiix",

	// Faction / Reputation
	"FactionTemplate": "niiiiiiiiiiiii",

	// Auction / Bank / Barber
	"AuctionHouse":      "niiixxxxxxxxxxxxxxxxx",
	"BankBagSlotPrices": "ni",
	"BarberShopStyle":   "nixxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxiii",
	"BattlemasterList":  "niiiiiiiiixssssssssssssssssxiixx",

	// Chat / Emotes
	"ChatChannels": "nixssssssssssssssssxxxxxxxxxxxxxxxxxx",
	"Emotes":       "nxxiiix",
	"EmotesText":   "nxixxxxxxxxxxxxxxxx",

	// Cinematic
	"CinematicCamera":    "nsiffff",
	"CinematicSequences": "nxixxxxxxx",

	// Currency / Destructible
	"CurrencyTypes":         "xnxi",
	"DestructibleModelData": "nxxixxxixxxixxxixxx",
	"DungeonEncounter":      "niixissssssssssssssssxx",
	"DurabilityCosts":       "niiiiiiiiiiiiiiiiiiiiiiiiiiiii",
	"DurabilityQuality":     "nf",

	// Holidays
	"Holidays": "niiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiixxsiix",

	// LFG
	"LFGDungeon": "nssssssssssssssssxiiiiiiiiixxixixxxxxxxxxxxxxxxxx",

	// Lock / Mail
	"Lock":         "niiiiiiiiiiiiiiiiiiiiiiiixxxxxxxx",
	"MailTemplate": "nxxxxxxxxxxxxxxxxxssssssssssssssssx",

	// Taxi
	"TaxiNodes":    "nifffssssssssssssssssxii",
	"TaxiPath":     "niii",
	"TaxiPathNode": "diiifffiiii",

	// Misc
	"Movie":               "nxx",
	"NamesReserved":       "xsx",
	"NamesProfanity":      "xsx",
	"OverrideSpellData":   "niiiiiiiiiix",
	"PowerDisplay":        "nixxxx",
	"QuestSort":           "nxxxxxxxxxxxxxxxxx",
	"QuestXP":             "niiiiiiiiii",
	"QuestFactionReward":  "niiiiiiiiii",
	"PvPDifficulty":       "diiiii",
	"RandomPropertiesPoints": "niiiiiiiiiiiiiii",
	"ScalingStatDistribution": "niiiiiiiiiiiiiiiiiiiii",
	"ScalingStatValues":   "iniiiiiiiiiiiiiiiiiiiiii",
	"SoundEntries":        "nxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	"StableSlotPrices":    "ni",
	"SummonProperties":    "niiiii",
	"TeamContributionPoints": "df",
	"TotemCategory":       "nxxxxxxxxxxxxxxxxxii",

	// Gems / Glyphs
	"GemProperties":   "nixxi",
	"GlyphProperties": "niix",
	"GlyphSlot":       "nii",
}

// Known LocalizedString fields: struct -> list of starting columns
// These are 16 consecutive 's' columns representing locale string offsets
var localizedFields = map[string][]int{
	"Achievement":           {4},
	"AchievementCriteria":   {},
	"AreaTable":             {11},
	"BattlemasterList":      {11},
	"BarberShopStyle":       {2},
	"ChatChannels":          {3},
	"CharTitles":            {2, 19},
	"ChrClasses":            {5},
	"ChrRaces":              {14},
	"CreatureFamily":        {10},
	"DungeonEncounter":      {5},
	"Faction":               {23},
	"ItemRandomProperties":  {7},
	"ItemRandomSuffix":      {1},
	"ItemSet":               {1},
	"LFGDungeon":            {1},
	"MailTemplate":          {18},
	"Map":                   {5},
	"SkillLine":             {3},
	"SpellEntry":            {136, 153},
	"TaxiNodes":             {4},
	"SpellRange":            {6},
}

// Known single *StringRef fields: struct -> list of columns
var stringRefFields = map[string][]int{
	"CinematicCamera":       {1},
	"GameObjectDisplayInfo": {1},
	"Holidays":              {51},
	"ItemDisplayInfo":       {5},
	"Map":                   {1},
}

// Field naming hints from AzerothCore DBCStructure.h
var fieldNames = map[string]map[int]string{
	"Achievement":       {0: "ID", 1: "RequiredFaction", 2: "MapID", 38: "CategoryId", 39: "Points", 41: "Flags", 60: "Count", 61: "RefAchievement"},
	"AchievementCategory": {0: "ID", 1: "ParentCategory"},
	"AchievementCriteria": {0: "ID", 1: "ReferredAchievement", 2: "RequiredType", 3: "Value1", 4: "Value2", 26: "Flags", 27: "TimedType", 28: "TimerStartEvent", 29: "TimeLimit"},
	"AreaGroup":          {0: "ID", 7: "NextGroup"},
	"AreaPOI":            {0: "ID", 12: "X", 13: "Y", 14: "Z", 15: "MapID", 17: "ZoneID", 52: "WorldState"},
	"AreaTable":          {0: "ID", 1: "MapID", 2: "Zone", 3: "ExploreFlag", 4: "Flags", 10: "AreaLevel", 28: "Team"},
	"AuctionHouse":       {0: "HouseID", 1: "Faction", 2: "DepositPercent", 3: "CutPercent"},
	"BankBagSlotPrices":  {0: "ID", 1: "Price"},
	"BarberShopStyle":    {0: "ID", 1: "Type", 37: "Race", 38: "Gender", 39: "HairID"},
	"BattlemasterList":   {0: "ID", 9: "Type", 28: "MaxGroupSize", 29: "HolidayID"},
	"CharTitles":         {0: "ID", 2: "NameMale", 19: "NameFemale", 36: "BitIndex"},
	"ChatChannels":       {0: "ID", 1: "Flags"},
	"ChrClasses":         {0: "ID", 2: "PowerType", 5: "Name", 56: "SpellFamily", 58: "CinematicSequence", 59: "Expansion"},
	"ChrRaces":           {0: "RaceID", 1: "Flags", 2: "FactionID", 4: "MaleDisplayID", 5: "FemaleDisplayID", 8: "BaseLanguage", 12: "CinematicSequence", 13: "Alliance", 14: "Name", 68: "Expansion"},
	"CinematicCamera":    {0: "ID", 2: "SoundID", 3: "OriginX", 4: "OriginY", 5: "OriginZ", 6: "OriginFacing"},
	"CinematicSequences": {0: "ID", 2: "CameraID"},
	"CreatureDisplayInfo": {0: "ID", 1: "ModelID", 2: "SoundID", 3: "ExtendedDisplayInfoID", 4: "Scale"},
	"CreatureDisplayInfoExtra": {0: "ID", 1: "DisplayRaceID"},
	"CreatureFamily":     {0: "ID", 1: "MinScale", 2: "MinScaleLevel", 3: "MaxScale", 4: "MaxScaleLevel", 5: "SkillLine1", 6: "SkillLine2", 7: "PetFoodMask", 8: "PetTalentType"},
	"CreatureModelData":  {0: "ID", 1: "Flags", 3: "ModelPath", 4: "Scale", 14: "CollisionWidth", 15: "CollisionHeight", 16: "MountHeight"},
	"CreatureSpellData":  {0: "ID"},
	"CurrencyTypes":      {1: "ItemID", 3: "BitIndex"},
	"DestructibleModelData": {0: "ID", 3: "DamagedDisplayID", 7: "DestroyedDisplayID", 11: "RebuildingDisplayID", 15: "SmokeDisplayID"},
	"DungeonEncounter":   {0: "ID", 1: "MapID", 2: "Difficulty", 4: "EncounterIndex"},
	"DurabilityCosts":    {0: "ID"},
	"DurabilityQuality":  {0: "ID", 1: "QualityMod"},
	"Emotes":             {0: "ID", 3: "Flags", 4: "EmoteType", 5: "UnitStandState"},
	"EmotesText":         {0: "ID", 1: "TextID"},
	"Faction":            {0: "ID", 1: "ReputationListID", 18: "Team", 19: "SpilloverRateIn", 20: "SpilloverRateOut", 21: "SpilloverMaxRankIn"},
	"FactionTemplate":    {0: "ID", 1: "Faction", 2: "FactionFlags", 3: "OurMask", 4: "FriendlyMask", 5: "HostileMask"},
	"GemProperties":      {0: "ID", 1: "EnchantmentID", 3: "Color"},
	"GlyphProperties":    {0: "ID", 1: "SpellID", 2: "TypeFlags"},
	"GlyphSlot":          {0: "ID", 1: "TypeFlags", 2: "Order"},
	"Holidays":           {0: "ID", 37: "Region", 38: "Looping", 52: "Priority", 53: "CalendarFilterType"},
	"Item":               {0: "ID", 1: "ClassID", 2: "SubclassID", 3: "SoundOverrideSubclassID", 4: "Material", 5: "DisplayInfoID", 6: "InventoryType", 7: "SheatheType"},
	"ItemExtendedCost":   {0: "ID", 1: "ReqHonorPoints", 2: "ReqArenaPoints", 3: "ReqArenaSlot", 15: "ReqPersonalArenaRating"},
	"ItemLimitCategory":  {0: "ID", 18: "MaxCount", 19: "Mode"},
	"LFGDungeon":         {0: "ID", 18: "MinLevel", 19: "MaxLevel", 20: "TargetLevel", 21: "TargetLevelMin", 22: "TargetLevelMax", 23: "MapID", 24: "Difficulty", 25: "Flags", 26: "TypeID", 29: "ExpansionLevel", 31: "GroupID"},
	"Light":              {0: "ID", 1: "MapID", 2: "X", 3: "Y", 4: "Z"},
	"LiquidType":         {0: "ID", 3: "Type", 5: "SpellID"},
	"Lock":               {0: "ID"},
	"Map":                {0: "ID", 2: "InstanceType", 3: "Flags", 22: "LinkedZone", 57: "MultimapID", 59: "EntranceMap", 60: "EntranceX", 61: "EntranceY", 63: "ExpansionID", 65: "MaxPlayers"},
	"MapDifficulty":      {0: "ID", 1: "MapID", 2: "Difficulty", 20: "ResetTime", 21: "MaxPlayers"},
	"Movie":              {0: "ID"},
	"OverrideSpellData":  {0: "ID"},
	"PowerDisplay":       {0: "ID", 1: "PowerType"},
	"PvPDifficulty":      {1: "MapID", 2: "BracketID", 3: "MinLevel", 4: "MaxLevel", 5: "Difficulty"},
	"QuestFactionReward": {0: "ID"},
	"QuestSort":          {0: "ID"},
	"QuestXP":            {0: "ID"},
	"RandomPropertiesPoints": {1: "ItemLevel"},
	"ScalingStatDistribution": {0: "ID", 21: "MaxLevel"},
	"ScalingStatValues":  {0: "ID", 1: "Level", 16: "SpellPower", 17: "SSDMultiplier2", 18: "SSDMultiplier3"},
	"SkillLine":          {0: "ID", 1: "CategoryID", 37: "SpellIcon", 55: "CanLink"},
	"SkillLineAbility":   {0: "ID", 1: "SkillLine", 2: "Spell", 3: "RaceMask", 4: "ClassMask", 7: "MinSkillLineRank", 8: "SupercededBySpell", 9: "AcquireMethod", 10: "TrivialSkillLineRankHigh", 11: "TrivialSkillLineRankLow"},
	"SkillRaceClassInfo": {1: "SkillID", 2: "RaceMask", 3: "ClassMask", 4: "Flags", 6: "SkillTierID"},
	"SkillTiers":         {0: "ID"},
	"SoundEntries":       {0: "ID"},
	"SpellEntry": {
		0: "Id", 1: "Category", 2: "Dispel", 3: "Mechanic",
		4: "Attributes", 5: "AttributesEx", 6: "AttributesEx2", 7: "AttributesEx3",
		8: "AttributesEx4", 9: "AttributesEx5", 10: "AttributesEx6", 11: "AttributesEx7",
		12: "Stances", 14: "StancesNot",
		16: "Targets", 17: "TargetCreatureType", 18: "RequiresSpellFocus", 19: "FacingCasterFlags",
		20: "CasterAuraState", 21: "TargetAuraState", 22: "CasterAuraStateNot", 23: "TargetAuraStateNot",
		24: "CasterAuraSpell", 25: "TargetAuraSpell", 26: "ExcludeCasterAuraSpell", 27: "ExcludeTargetAuraSpell",
		28: "CastingTimeIndex", 29: "RecoveryTime", 30: "CategoryRecoveryTime", 31: "InterruptFlags",
		32: "AuraInterruptFlags", 33: "ChannelInterruptFlags", 34: "ProcFlags", 35: "ProcChance",
		36: "ProcCharges", 37: "MaxLevel", 38: "BaseLevel", 39: "SpellLevel",
		40: "DurationIndex", 41: "PowerType", 42: "ManaCost", 43: "ManaCostPerlevel",
		44: "ManaPerSecond", 45: "ManaPerSecondPerLevel", 46: "RangeIndex", 47: "Speed",
		49: "StackAmount",
		50: "Totem0", 51: "Totem1",
		52: "Reagent0", 53: "Reagent1", 54: "Reagent2", 55: "Reagent3",
		56: "Reagent4", 57: "Reagent5", 58: "Reagent6", 59: "Reagent7",
		60: "ReagentCount0", 61: "ReagentCount1", 62: "ReagentCount2", 63: "ReagentCount3",
		64: "ReagentCount4", 65: "ReagentCount5", 66: "ReagentCount6", 67: "ReagentCount7",
		68: "EquippedItemClass", 69: "EquippedItemSubClassMask", 70: "EquippedItemInventoryTypeMask",
		71: "Effect0", 72: "Effect1", 73: "Effect2",
		74: "EffectDieSides0", 75: "EffectDieSides1", 76: "EffectDieSides2",
		77: "EffectRealPointsPerLevel0", 78: "EffectRealPointsPerLevel1", 79: "EffectRealPointsPerLevel2",
		80: "EffectBasePoints0", 81: "EffectBasePoints1", 82: "EffectBasePoints2",
		83: "EffectMechanic0", 84: "EffectMechanic1", 85: "EffectMechanic2",
		86: "EffectImplicitTargetA0", 87: "EffectImplicitTargetA1", 88: "EffectImplicitTargetA2",
		89: "EffectImplicitTargetB0", 90: "EffectImplicitTargetB1", 91: "EffectImplicitTargetB2",
		92: "EffectRadiusIndex0", 93: "EffectRadiusIndex1", 94: "EffectRadiusIndex2",
		95: "EffectApplyAuraName0", 96: "EffectApplyAuraName1", 97: "EffectApplyAuraName2",
		98: "EffectAmplitude0", 99: "EffectAmplitude1", 100: "EffectAmplitude2",
		101: "EffectValueMultiplier0", 102: "EffectValueMultiplier1", 103: "EffectValueMultiplier2",
		104: "EffectChainTarget0", 105: "EffectChainTarget1", 106: "EffectChainTarget2",
		107: "EffectItemType0", 108: "EffectItemType1", 109: "EffectItemType2",
		110: "EffectMiscValue0", 111: "EffectMiscValue1", 112: "EffectMiscValue2",
		113: "EffectMiscValueB0", 114: "EffectMiscValueB1", 115: "EffectMiscValueB2",
		116: "EffectTriggerSpell0", 117: "EffectTriggerSpell1", 118: "EffectTriggerSpell2",
		119: "EffectPointsPerComboPoint0", 120: "EffectPointsPerComboPoint1", 121: "EffectPointsPerComboPoint2",
		122: "EffectSpellClassMask0_0", 123: "EffectSpellClassMask0_1", 124: "EffectSpellClassMask0_2",
		125: "EffectSpellClassMask1_0", 126: "EffectSpellClassMask1_1", 127: "EffectSpellClassMask1_2",
		128: "EffectSpellClassMask2_0", 129: "EffectSpellClassMask2_1", 130: "EffectSpellClassMask2_2",
		131: "SpellVisual0", 132: "SpellVisual1",
		133: "SpellIconID", 134: "ActiveIconID", 135: "SpellPriority",
		204: "ManaCostPercentage", 205: "StartRecoveryCategory", 206: "StartRecoveryTime",
		207: "MaxTargetLevel", 208: "SpellFamilyName",
		209: "SpellFamilyFlags0", 210: "SpellFamilyFlags1", 211: "SpellFamilyFlags2",
		212: "MaxAffectedTargets", 213: "DmgClass", 214: "PreventionType",
		216: "EffectDamageMultiplier0", 217: "EffectDamageMultiplier1", 218: "EffectDamageMultiplier2",
		222: "TotemCategory0", 223: "TotemCategory1",
		224: "AreaGroupId", 225: "SchoolMask", 226: "RuneCostID",
		229: "EffectBonusMultiplier0", 230: "EffectBonusMultiplier1", 231: "EffectBonusMultiplier2",
	},
	"SpellCastTime":      {0: "ID", 1: "CastTime"},
	"SpellCategory":      {0: "ID", 1: "Flags"},
	"SpellDifficulty":    {0: "ID"},
	"SpellDuration":      {0: "ID", 1: "Duration", 2: "PerLevel", 3: "MaxDuration"},
	"SpellFocusObject":   {0: "ID"},
	"SpellItemEnchantment": {0: "ID", 28: "ItemDisplayInfoID"},
	"SpellRadius":        {0: "ID", 1: "RadiusMin", 2: "RadiusPerLevel", 3: "RadiusMax"},
	"SpellRange":         {0: "ID", 5: "Flags"},
	"SpellRuneCost":      {0: "ID", 4: "RunePowerGain"},
	"SpellShapeshiftForm": {0: "ID", 18: "Flags", 20: "CreatureType", 22: "AttackIcon"},
	"SpellVisual":        {0: "ID", 7: "PrecastKit"},
	"StableSlotPrices":   {0: "ID", 1: "Price"},
	"SummonProperties":   {0: "ID", 1: "Flags", 2: "FactionID", 3: "Type", 4: "Slot"},
	"Talent":             {0: "ID", 1: "TabID", 2: "ClassID", 3: "Row", 4: "Col", 5: "SpellID", 6: "AltMask", 7: "DependsOn"},
	"TalentTab":          {0: "ID", 19: "ClassMask", 20: "PetTalentMask", 21: "OrderIndex"},
	"TaxiNodes":          {0: "ID", 1: "X", 2: "Y", 3: "Z", 21: "MountCreatureID1", 22: "MountCreatureID2"},
	"TaxiPath":           {0: "ID", 1: "From", 2: "To", 3: "Price"},
	"TaxiPathNode":       {0: "ID", 1: "PathID", 2: "NodeIndex", 3: "MapID", 4: "X", 5: "Y", 6: "Z", 7: "Delay", 8: "ArriveEventID", 9: "DepartEventID"},
	"TeamContributionPoints": {0: "Value"},
	"TotemCategory":      {0: "ID", 17: "TotemCategoryType", 18: "TotemCategoryMask"},
	"TransportAnimation": {0: "ID", 1: "TransportID", 2: "TimeSeg", 3: "X", 4: "Y", 5: "Z"},
	"TransportRotation":  {0: "ID", 1: "TransportID", 2: "TimeSeg", 3: "X", 4: "Y", 5: "Z", 6: "W"},
	"Vehicle":            {0: "ID", 1: "Flags", 2: "TurnSpeed", 3: "PitchSpeed", 4: "PitchMin", 5: "PitchMax", 6: "SeatID"},
	"VehicleSeat":        {0: "ID", 1: "Flags", 2: "AttachmentID", 3: "AttachmentOffset"},
	"WMOAreaTable":       {0: "ID", 1: "WMOID", 2: "NameIndex", 3: "ZoneID", 9: "GroupID", 10: "AreaID"},
	"WorldMapArea":       {2: "MapID", 4: "Left", 5: "Right", 6: "Top", 7: "Bottom"},
	"WorldMapOverlay":    {0: "ID", 2: "MapID", 3: "AreaID", 4: "AreaID2", 5: "AreaID3"},
	"GameObjectArtKit":   {0: "ID"},
	"GameObjectDisplayInfo": {0: "ID", 7: "MinX", 8: "MinY", 9: "MinZ", 10: "MaxX", 11: "MaxY", 12: "MaxZ"},
}

// File grouping: filename (without .go) -> list of DBC names
var fileGroups = map[string][]string{
	"achievement": {"Achievement", "AchievementCategory", "AchievementCriteria"},
	// character is handled manually in character.go (needs special types)
	"area":        {"AreaTable", "AreaGroup", "AreaPOI", "WMOAreaTable", "WorldMapArea", "WorldMapOverlay"},
	"creature":    {"CreatureDisplayInfo", "CreatureDisplayInfoExtra", "CreatureFamily", "CreatureModelData", "CreatureSpellData", "CreatureType"},
	"emotes":      {"Emotes", "EmotesText"},
	"faction":     {"Faction", "FactionTemplate"},
	"gameobject":  {"GameObjectArtKit", "GameObjectDisplayInfo"},
	"gt":          {"GtBarberShopCostBase", "GtChanceToMeleeCrit", "GtChanceToMeleeCritBase", "GtChanceToSpellCrit", "GtChanceToSpellCritBase", "GtCombatRatings", "GtNPCManaCostScaler", "GtOCTClassCombatRatingScalar", "GtOCTRegenHP", "GtRegenHPPerSpt", "GtRegenMPPerSpt"},
	"item":        {"Item", "ItemBagFamily", "ItemDisplayInfo", "ItemExtendedCost", "ItemLimitCategory", "ItemRandomProperties", "ItemRandomSuffix", "ItemSet"},
	"lfg":         {"LFGDungeon", "DungeonEncounter"},
	"liquid":      {"LiquidType", "Light"},
	"lock":        {"Lock"},
	"mail":        {"MailTemplate"},
	"map":         {"Map", "MapDifficulty"},
	"misc":        {"BankBagSlotPrices", "BarberShopStyle", "BattlemasterList", "ChatChannels", "CinematicCamera", "CinematicSequences", "CurrencyTypes", "DestructibleModelData", "DurabilityCosts", "DurabilityQuality", "GemProperties", "GlyphProperties", "GlyphSlot", "Holidays", "Movie", "NamesProfanity", "NamesReserved", "OverrideSpellData", "PowerDisplay", "PvPDifficulty", "QuestFactionReward", "QuestSort", "QuestXP", "RandomPropertiesPoints", "ScalingStatDistribution", "ScalingStatValues", "SoundEntries", "StableSlotPrices", "SummonProperties", "TeamContributionPoints", "TotemCategory", "TransportAnimation", "TransportRotation"},
	"skill":       {"SkillLine", "SkillLineAbility", "SkillRaceClassInfo", "SkillTiers", "Talent", "TalentTab"},
	"spell":       {"SpellEntry", "SpellCastTime", "SpellCategory", "SpellDifficulty", "SpellDuration", "SpellFocusObject", "SpellRadius", "SpellRange", "SpellRuneCost", "SpellItemEnchantment", "SpellItemEnchantmentCondition", "SpellShapeshiftForm", "SpellVisual"},
	"taxi":        {"TaxiNodes", "TaxiPath", "TaxiPathNode"},
	"vehicle":     {"Vehicle", "VehicleSeat"},
}

// Format: 'i' -> 'i', 'f' -> 'f', 's' -> 's', 'x' -> 'x', 'X' -> 'X', 'b' -> 'b', 'n' -> 'n', 'd' -> 'd', 'l' -> 'l'
func isLocalizedString(structName string, col int) bool {
	cols := localizedFields[structName]
	for _, c := range cols {
		if col == c {
			return true
		}
	}
	return false
}

func isStringRef(structName string, col int) bool {
	cols := stringRefFields[structName]
	for _, c := range cols {
		if col == c {
			return true
		}
	}
	return false
}

func getFieldName(structName string, col int) string {
	if names, ok := fieldNames[structName]; ok {
		if name, ok := names[col]; ok {
			return name
		}
	}
	return ""
}

func toGoType(ch byte) string {
	switch ch {
	case 'i', 'n', 'd', 'l':
		return "uint32"
	case 'f':
		return "float32"
	case 'b':
		return "uint8"
	case 's':
		return "uint32" // string offset
	default:
		return "uint32"
	}
}

func exportName(s string) string {
	var result strings.Builder
	upper := true
	for _, r := range s {
		if r == '_' || r == ' ' {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func generateStruct(name string, format string) string {
	var sb strings.Builder

	structName := name + "Entry"

	sb.WriteString(fmt.Sprintf("// %s represents the %s.dbc file structure.\n", structName, name))
	sb.WriteString(fmt.Sprintf("//\n// Format: %s\n", format))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	col := 0
	i := 0
	for i < len(format) {
		ch := format[i]

		switch ch {
		case 'n', 'd':
			// Index field
			fieldName := getFieldName(name, col)
			if fieldName == "" {
				fieldName = fmt.Sprintf("Index%d", col)
			}
			sb.WriteString(fmt.Sprintf("\t%s uint32 `dbc:\"offset=%d\"`\n", fieldName, col))
			col++
			i++

		case 'i':
			fieldName := getFieldName(name, col)
			if fieldName == "" {
				fieldName = fmt.Sprintf("Field%d", col)
			}
			sb.WriteString(fmt.Sprintf("\t%s uint32 `dbc:\"offset=%d\"`\n", fieldName, col))
			col++
			i++

		case 'f':
			fieldName := getFieldName(name, col)
			if fieldName == "" {
				fieldName = fmt.Sprintf("Field%d", col)
			}
			sb.WriteString(fmt.Sprintf("\t%s float32 `dbc:\"offset=%d\"`\n", fieldName, col))
			col++
			i++

		case 's':
			if isLocalizedString(name, col) {
				fieldName := getFieldName(name, col)
				if fieldName == "" {
					fieldName = fmt.Sprintf("Field%d", col)
				}
				sb.WriteString(fmt.Sprintf("\t%s LocalizedString `dbc:\"offset=%d\"`\n", fieldName, col))
				col += 16
				i += 16
			} else if isStringRef(name, col) {
				fieldName := getFieldName(name, col)
				if fieldName == "" {
					fieldName = fmt.Sprintf("Field%d", col)
				}
				sb.WriteString(fmt.Sprintf("\t%s *StringRef `dbc:\"offset=%d\"`\n", fieldName, col))
				col++
				i++
			} else {
				fieldName := getFieldName(name, col)
				if fieldName == "" {
					fieldName = fmt.Sprintf("Field%d", col)
				}
				sb.WriteString(fmt.Sprintf("\t%s uint32 `dbc:\"offset=%d\"` // string offset\n", fieldName, col))
				col++
				i++
			}

		case 'x':
			col++
			i++

		case 'X':
			i++

		case 'b':
			fieldName := getFieldName(name, col)
			if fieldName == "" {
				fieldName = fmt.Sprintf("Field%d", col)
			}
			sb.WriteString(fmt.Sprintf("\t%s uint8 `dbc:\"offset=%d\"`\n", fieldName, col))
			col++
			i++

		default:
			i++
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

func main() {
	outDir := filepath.Join("pkg", "summit", "tools", "dbc", "wotlk")

	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Sort file group names for deterministic output
	groupNames := make([]string, 0, len(fileGroups))
	for name := range fileGroups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	for _, group := range groupNames {
		dbcNames := fileGroups[group]
		sort.Strings(dbcNames)

		var sb strings.Builder
		sb.WriteString("// Code generated by dbc-gen from AzerothCore DBCfmt.h. DO NOT EDIT!\n")
		sb.WriteString("//\n")
		sb.WriteString("// DBC file format for World of Warcraft 3.3.5a (WotLK).\n")
		sb.WriteString("// Each field uses a dbc tag specifying its column offset in the DBC record.\n")
		sb.WriteString("// LocalizedString fields occupy 16 columns (16 locale string offsets).\n")
		sb.WriteString("// *StringRef fields are single string offsets into the string block.\n\n")
		sb.WriteString("package wotlk\n\n")

		for _, dbcName := range dbcNames {
			format, ok := dbcs[dbcName]
			if !ok {
				fmt.Fprintf(os.Stderr, "Warning: no format for %s\n", dbcName)
				continue
			}
			sb.WriteString(generateStruct(dbcName, format))
			sb.WriteString("\n")
		}

		outFile := filepath.Join(outDir, group+".go")
		if err := os.WriteFile(outFile, []byte(sb.String()), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outFile, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s.go\n", group)
	}

	fmt.Println("Done.")
}
