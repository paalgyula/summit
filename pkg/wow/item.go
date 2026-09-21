package wow

// ItemClass represents the class of an item (ITEM_CLASS_*).
type ItemClass uint32

const (
	ItemClassConsumable  ItemClass = 0
	ItemClassContainer   ItemClass = 1
	ItemClassWeapon      ItemClass = 2
	ItemClassGem         ItemClass = 3
	ItemClassArmor       ItemClass = 4
	ItemClassReagent     ItemClass = 5
	ItemClassProjectile  ItemClass = 6
	ItemClassTradeGoods  ItemClass = 7
	ItemClassGeneric     ItemClass = 8
	ItemClassRecipe      ItemClass = 9
	ItemClassMoney       ItemClass = 10
	ItemClassQuiver      ItemClass = 11
	ItemClassQuest       ItemClass = 12
	ItemClassKey         ItemClass = 13
	ItemClassPermanent   ItemClass = 14
	ItemClassMisc        ItemClass = 15
	ItemClassGlyph       ItemClass = 16
)

// ItemSubclassWeapon represents weapon subtypes.
type ItemSubclassWeapon uint32

const (
	WeaponSubclassAxe       ItemSubclassWeapon = 0
	WeaponSubclassAxe2H     ItemSubclassWeapon = 1
	WeaponSubclassBow       ItemSubclassWeapon = 2
	WeaponSubclassGun       ItemSubclassWeapon = 3
	WeaponSubclassMace      ItemSubclassWeapon = 4
	WeaponSubclassMace2H    ItemSubclassWeapon = 5
	WeaponSubclassPolearm   ItemSubclassWeapon = 6
	WeaponSubclassSword     ItemSubclassWeapon = 7
	WeaponSubclassSword2H   ItemSubclassWeapon = 8
	WeaponSubclassStaff     ItemSubclassWeapon = 10
	WeaponSubclassExotic    ItemSubclassWeapon = 11
	WeaponSubclassExotic2   ItemSubclassWeapon = 12
	WeaponSubclassFist      ItemSubclassWeapon = 13
	WeaponSubclassDagger    ItemSubclassWeapon = 15
	WeaponSubclassThrown    ItemSubclassWeapon = 16
	WeaponSubclassCrossbow  ItemSubclassWeapon = 18
	WeaponSubclassWand      ItemSubclassWeapon = 19
	WeaponSubclassFishingPole ItemSubclassWeapon = 20
)

// ItemSubclassArmor represents armor subtypes.
type ItemSubclassArmor uint32

const (
	ArmorSubclassMisc     ItemSubclassArmor = 0
	ArmorSubclassCloth    ItemSubclassArmor = 1
	ArmorSubclassLeather  ItemSubclassArmor = 2
	ArmorSubclassMail     ItemSubclassArmor = 3
	ArmorSubclassPlate    ItemSubclassArmor = 4
	ArmorSubclassBuckler  ItemSubclassArmor = 5
	ArmorSubclassShield   ItemSubclassArmor = 6
	ArmorSubclassLibram   ItemSubclassArmor = 7
	ArmorSubclassIdol     ItemSubclassArmor = 8
	ArmorSubclassTotem    ItemSubclassArmor = 9
	ArmorSubclassSigil    ItemSubclassArmor = 10
)

// ItemFlag represents item flags (ITEM_FLAG_*).
type ItemFlag uint32

const (
	ItemFlagUnknown0         ItemFlag = 0x00000001
	ItemFlagConjured         ItemFlag = 0x00000002
	ItemFlagLooted           ItemFlag = 0x00000004
	ItemFlagItemInUse        ItemFlag = 0x00000008
	ItemFlagUnknown4         ItemFlag = 0x00000010
	ItemFlagUnknown5         ItemFlag = 0x00000020
	ItemFlagUnknown6         ItemFlag = 0x00000040
	ItemFlagUnknown7         ItemFlag = 0x00000080
	ItemFlagUnknown8         ItemFlag = 0x00000100
	ItemFlagUnknown9         ItemFlag = 0x00000200
	ItemFlagUnknown10        ItemFlag = 0x00000400
	ItemFlagUnknown11        ItemFlag = 0x00000800
	ItemFlagUnknown12        ItemFlag = 0x00001000
	ItemFlagUnknown13        ItemFlag = 0x00002000
	ItemFlagUnknown14        ItemFlag = 0x00004000
	ItemFlagUnknown15        ItemFlag = 0x00008000
	ItemFlagNoPickup         ItemFlag = 0x00010000
	ItemFlagUnknown17        ItemFlag = 0x00020000
	ItemFlagUnknown18        ItemFlag = 0x00040000
	ItemFlagUnknown19        ItemFlag = 0x00080000
	ItemFlagUnknown20        ItemFlag = 0x00100000
	ItemFlagUnknown21        ItemFlag = 0x00200000
	ItemFlagUnknown22        ItemFlag = 0x00400000
	ItemFlagUnknown23        ItemFlag = 0x00800000
	ItemFlagUnknown24        ItemFlag = 0x01000000
	ItemFlagUnknown25        ItemFlag = 0x02000000
	ItemFlagUnknown26        ItemFlag = 0x04000000
	ItemFlagNoEquipCooldown  ItemFlag = 0x08000000
	ItemFlagUnknown28        ItemFlag = 0x10000000
	ItemFlagNoBuyForNPC      ItemFlag = 0x20000000
	ItemFlagUnknown30        ItemFlag = 0x40000000
	ItemFlagNoCreator        ItemFlag = 0x80000000
)

// ItemBonding represents how an item binds.
type ItemBonding uint32

const (
	BondingNoBind     ItemBonding = 0
	BondingOnPickup   ItemBonding = 1
	BondingOnEquip    ItemBonding = 2
	BondingOnUse      ItemBonding = 3
	BondingOnQuest    ItemBonding = 4
)

// ItemQuality represents item quality levels.
type ItemQuality uint32

const (
	QualityPoor        ItemQuality = 0
	QualityCommon      ItemQuality = 1
	QualityUncommon    ItemQuality = 2
	QualityRare        ItemQuality = 3
	QualityEpic        ItemQuality = 4
	QualityLegendary   ItemQuality = 5
	QualityHeirloom    ItemQuality = 6
)

// Socket colors.
const (
	SocketColorMeta     = 1
	SocketColorRed      = 2
	SocketColorYellow   = 4
	SocketColorBlue     = 8
)

// Item stat types (ITEM_STAT_TYPE_*).
const (
	Mana          = 0
	Health        = 1
	Agility       = 3
	Strength      = 4
	Intellect     = 5
	Spirit        = 6
	Stamina       = 7
	AttackPower   = 38
	RangedAttackPower = 39
)

// Binding type for items (used in WoW 3.3.5a).
const (
	ItemBindOnPickup = 1
	ItemBindOnEquip  = 2
	ItemBindOnUse    = 3
	ItemBindOnQuest  = 4
)
