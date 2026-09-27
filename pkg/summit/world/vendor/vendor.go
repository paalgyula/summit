package vendor

import "github.com/paalgyula/summit/pkg/store"

// NPC flags relating to vendors (matching AzerothCore / 3.3.5a protocol).
const (
	NpcFlagVendor        uint32 = 0x00000080
	NpcFlagVendorAmmo    uint32 = 0x00000100
	NpcFlagVendorFood    uint32 = 0x00000200
	NpcFlagVendorPoison  uint32 = 0x00000400
	NpcFlagVendorReagent uint32 = 0x00000800
	NpcFlagRepair        uint32 = 0x00001000

	VendorMask uint32 = NpcFlagVendor |
		NpcFlagVendorAmmo |
		NpcFlagVendorFood |
		NpcFlagVendorPoison |
		NpcFlagVendorReagent |
		NpcFlagRepair
)

// GossipOption types matching AzerothCore's Gossip_Option enum.
const (
	GossipOptionNone        uint32 = 0
	GossipOptionGossip      uint32 = 1
	GossipOptionQuestGiver  uint32 = 2
	GossipOptionVendor      uint32 = 3
	GossipOptionTaxiVendor  uint32 = 4
	GossipOptionTrainer     uint32 = 5
	GossipOptionSpiritHeal  uint32 = 6
	GossipOptionInnkeeper   uint32 = 8
	GossipOptionBanker      uint32 = 9
	GossipOptionPetitioner  uint32 = 10
	GossipOptionTabard      uint32 = 11
	GossipOptionBattlefield uint32 = 12
	GossipOptionAuctioneer  uint32 = 13
	GossipOptionStablePet   uint32 = 14
	GossipOptionArmorer     uint32 = 15
)

// GossipOption icons matching AzerothCore's GossipOptionIcon enum.
const (
	GossipIconChat      uint8 = 0 // white chat bubble
	GossipIconVendor    uint8 = 1 // brown bag / pouch
	GossipIconTaxi      uint8 = 2 // flightmarker
	GossipIconTrainer   uint8 = 3 // brown book
	GossipIconInteract1 uint8 = 4 // golden cog
	GossipIconInteract2 uint8 = 5 // golden cog
	GossipIconMoneyBag  uint8 = 6 // brown bag with gold coin
	GossipIconTalk      uint8 = 7 // white chat bubble with "..."
	GossipIconTabard    uint8 = 8 // tabard
	GossipIconBattle    uint8 = 9 // crossed swords
	GossipIconDot       uint8 = 10
)

// BuyResult matches AzerothCore's BuyResult enum from Item.h.
type BuyResult uint8

const (
	BuyErrCantFindItem       BuyResult = 0
	BuyErrItemAlreadySold    BuyResult = 1
	BuyErrNotEnoughMoney     BuyResult = 2
	BuyErrSellerDontLikeYou  BuyResult = 4
	BuyErrDistanceTooFar     BuyResult = 5
	BuyErrItemSoldOut        BuyResult = 7
	BuyErrCantCarryMore      BuyResult = 8
	BuyErrRankRequire        BuyResult = 11
	BuyErrReputationRequire  BuyResult = 12
)

// SellResult matches AzerothCore's SellResult enum from Item.h.
type SellResult uint8

const (
	SellErrCantFindItem                 SellResult = 1 // The item was not found.
	SellErrCantSellItem                 SellResult = 2 // The merchant doesn't want that item.
	SellErrCantFindVendor               SellResult = 3 // The merchant doesn't like you / vendor not found.
	SellErrYouDontOwnThatItem           SellResult = 4 // You don't own that item.
	SellErrUnk                          SellResult = 5 // Nothing appears...
	SellErrOnlyEmptyBag                 SellResult = 6 // You can only do that with empty bags.
	SellErrCantSellToThisMerchant       SellResult = 7 // You cannot sell items to this merchant.
	SellErrMustRepairItemDurabilityToUse SellResult = 8 // You must repair that item's durability.
	SellInternalBagError                SellResult = 9 // Internal Bag Error.
)

// MaxVendorItems is the maximum number of items a vendor can show in SMSG_LIST_INVENTORY (matches AC MAX_VENDOR_ITEMS = 150).
const MaxVendorItems = 150

// Buyback slots in 3.3.5a (matches AC BUYBACK_SLOT_START=74, BUYBACK_SLOT_END=86).
const (
	BuybackSlotStart = 74
	BuybackSlotEnd   = 86
	BuybackSlotCount = 12
)

// IsVendor returns true if the NPC flags include any vendor service flag.
func IsVendor(npcFlags uint32) bool {
	return (npcFlags & VendorMask) != 0
}

// VendorItemAlias is an alias to store.VendorItem.
type VendorItem = store.VendorItem
