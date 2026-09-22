package model

import (
	"math/big"
	"time"

	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
)

// --- Account transforms ---

// AccountToEntity converts a domain Account to a MongoDB entity.
func AccountToEntity(acc *store.Account) AccountEntity {
	e := AccountEntity{
		Name:      acc.Name,
		Email:     acc.Email,
		CreatedAt: acc.CreatedAt,
		LastLogin: acc.LastLogin,
		Activated: acc.Activated,
	}

	if acc.Verifier != nil {
		e.Verifier = acc.Verifier.Text(16)
	}

	if acc.Salt != nil {
		e.Salt = acc.Salt.Text(16)
	}

	if acc.Ban != nil {
		e.BanReason = acc.Ban.BanReason
		e.BannedBy = acc.Ban.BannedBy

		if t, err := time.Parse(time.RFC3339, acc.Ban.BannedAt); err == nil {
			e.BannedAt = &t
		}

		e.BanExpiry = acc.Ban.Expires
	}

	return e
}

// EntityToAccount converts a MongoDB entity to a domain Account.
func EntityToAccount(e AccountEntity) (*store.Account, error) {
	acc := &store.Account{
		ID:        e.IDString,
		Name:      e.Name,
		Email:     e.Email,
		CreatedAt: e.CreatedAt,
		LastLogin: e.LastLogin,
		Activated: e.Activated,
	}

	if e.Verifier != "" {
		v, ok := new(big.Int).SetString(e.Verifier, 16)
		if !ok {
			return nil, store.ErrInvalidHexNumber
		}

		acc.Verifier = v
	}

	if e.Salt != "" {
		s, ok := new(big.Int).SetString(e.Salt, 16)
		if !ok {
			return nil, store.ErrInvalidHexNumber
		}

		acc.Salt = s
	}

	if e.BanReason != "" || e.BannedAt != nil {
		acc.Ban = &store.AccountBan{
			BanReason: e.BanReason,
			BannedBy:  e.BannedBy,
			Expires:   e.BanExpiry,
		}

		if e.BannedAt != nil {
			acc.Ban.BannedAt = e.BannedAt.Format(time.RFC3339)
		}
	}

	return acc, nil
}

// --- Character transforms ---

// PlayerToEntity converts a domain Player to a MongoDB entity.
// The account parameter is the uppercase account name this character belongs to.
func PlayerToEntity(p *player.Player, account string) CharacterEntity {
	now := time.Now()

	e := CharacterEntity{
		GUID:       p.ID,
		Account:    account,
		Name:       p.Name,
		Race:       uint8(p.Race),
		Class:      uint8(p.Class),
		Gender:     uint8(p.Gender),
		Skin:       p.Skin,
		Face:       p.Face,
		HairStyle:  p.HairStyle,
		HairColor:  p.HairColor,
		FacialHair: p.FacialHair,
		OutfitID:   p.OutfitID,

		Location: LocationEntity{
			Map:  p.Location.Map,
			Zone: p.Location.Zone,
			X:    p.Location.X,
			Y:    p.Location.Y,
			Z:    p.Location.Z,
			O:    p.Location.O,
		},
		BindLocation: LocationEntity{
			Map:  p.BindLocation.Map,
			Zone: p.BindLocation.Zone,
			X:    p.BindLocation.X,
			Y:    p.BindLocation.Y,
			Z:    p.BindLocation.Z,
			O:    p.BindLocation.O,
		},

		Level: p.Level,
		XP:    p.XP,
		Money: p.Money,

		Health:    p.Health,
		MaxHealth: p.MaxHealth,
		Power:     p.Power[:],
		MaxPower:  p.MaxPower[:],

		DisplayID:       p.DisplayID,
		NativeDisplayID: p.NativeDisplayID,

		GuildID:         p.GuildID,
		CharFlags:       p.CharFlags,
		PlayerFlags:     uint32(p.PlayerFlags),
		Recustomization: p.Recustomization,
		FirstLogin:      p.FirstLogin,

		Pet: PetEntity{
			DisplayID: p.Pet.DisplayID,
			Level:     p.Pet.PetLevel,
			Family:    p.Pet.PetFamilly,
		},

		KnownSpells: p.KnownSpells,
		Actions:     p.Actions[:],

		CreatedAt: now,
		UpdatedAt: now,
	}

	// Inventory
	if p.Inventory != nil {
		e.Inventory = make([]ItemEntity, 0)

		for i := 0; i < player.InventorySlotTotal; i++ {
			item := p.Inventory.GetItem(i)
			if item == nil {
				continue
			}

			e.Inventory = append(e.Inventory, ItemToEntity(item, uint16(i)))
		}
	}

	// Quests (embedded)
	e.Quests, e.RewardedQuests = QuestProgressToEntity(p)

	return e
}

// QuestProgressToEntity extracts the character's quest progress for embedding in
// the character document. It is also used for partial (`$set`) quest updates.
func QuestProgressToEntity(p *player.Player) ([]QuestProgressEntity, []uint32) {
	var quests []QuestProgressEntity

	if qs, ok := p.QuestStatus.(map[uint32]*quest.QuestStatusData); ok && len(qs) > 0 {
		quests = make([]QuestProgressEntity, 0, len(qs))

		for id, s := range qs {
			if s == nil {
				continue
			}

			quests = append(quests, QuestProgressEntity{
				QuestID:           id,
				Status:            uint32(s.Status),
				Timer:             s.Timer,
				ItemCount:         s.ItemCount,
				CreatureOrGOCount: s.CreatureOrGOCount,
				PlayerCount:       s.PlayerCount,
				Explored:          s.Explored,
			})
		}
	}

	var rewarded []uint32

	if rw, ok := p.RewardedQuests.(map[uint32]bool); ok && len(rw) > 0 {
		rewarded = make([]uint32, 0, len(rw))

		for id := range rw {
			rewarded = append(rewarded, id)
		}
	}

	return quests, rewarded
}

// EntityToPlayer converts a MongoDB entity to a domain Player.
func EntityToPlayer(e CharacterEntity) *player.Player {
	p := player.NewPlayer()

	p.ID = e.GUID
	p.Name = e.Name
	p.Race = wow.PlayerRace(e.Race)
	p.Class = wow.PlayerClass(e.Class)
	p.Gender = wow.PlayerGender(e.Gender)
	p.FactionID = player.FactionTemplateFromRace(p.Race)

	p.Skin = e.Skin
	p.Face = e.Face
	p.HairStyle = e.HairStyle
	p.HairColor = e.HairColor
	p.FacialHair = e.FacialHair
	p.OutfitID = e.OutfitID

	p.Location = player.WorldLocation{
		Map:  e.Location.Map,
		Zone: e.Location.Zone,
		X:    e.Location.X,
		Y:    e.Location.Y,
		Z:    e.Location.Z,
		O:    e.Location.O,
	}
	p.BindLocation = player.WorldLocation{
		Map:  e.BindLocation.Map,
		Zone: e.BindLocation.Zone,
		X:    e.BindLocation.X,
		Y:    e.BindLocation.Y,
		Z:    e.BindLocation.Z,
		O:    e.BindLocation.O,
	}

	p.Level = e.Level
	p.XP = e.XP
	p.Money = e.Money

	p.Health = e.Health
	p.MaxHealth = e.MaxHealth

	if len(e.Power) > 0 {
		copy(p.Power[:], e.Power)
	}

	if len(e.MaxPower) > 0 {
		copy(p.MaxPower[:], e.MaxPower)
	}

	p.DisplayID = e.DisplayID
	p.NativeDisplayID = e.NativeDisplayID

	p.GuildID = e.GuildID
	p.CharFlags = e.CharFlags
	p.PlayerFlags = wow.PlayerFlag(e.PlayerFlags)
	p.Recustomization = e.Recustomization
	p.FirstLogin = e.FirstLogin

	p.Pet = player.Pet{
		DisplayID:  e.Pet.DisplayID,
		PetLevel:   e.Pet.Level,
		PetFamilly: e.Pet.Family,
	}

	p.KnownSpells = e.KnownSpells

	if len(e.Actions) > 0 {
		copy(p.Actions[:], e.Actions)
	}

	// Quests (embedded)
	if len(e.Quests) > 0 {
		status := make(map[uint32]*quest.QuestStatusData, len(e.Quests))

		for _, qe := range e.Quests {
			status[qe.QuestID] = &quest.QuestStatusData{
				Status:            quest.QuestStatus(qe.Status),
				Timer:             qe.Timer,
				ItemCount:         qe.ItemCount,
				CreatureOrGOCount: qe.CreatureOrGOCount,
				PlayerCount:       qe.PlayerCount,
				Explored:          qe.Explored,
			}
		}

		p.QuestStatus = status
	}

	if len(e.RewardedQuests) > 0 {
		rewarded := make(map[uint32]bool, len(e.RewardedQuests))

		for _, id := range e.RewardedQuests {
			rewarded[id] = true
		}

		p.RewardedQuests = rewarded
	}

	// Inventory
	if len(e.Inventory) > 0 {
		p.Inventory = player.NewInventory()

		for _, ie := range e.Inventory {
			item := EntityToItem(ie)
			p.Inventory.SetItem(int(ie.Slot), item)
		}
	}

	return p
}

// --- Item transforms ---

// ItemToEntity converts a domain Item to a MongoDB entity.
func ItemToEntity(item *player.Item, slot uint16) ItemEntity {
	return ItemEntity{
		Slot:       slot,
		Entry:      item.ItemEntry,
		Count:      item.StackCount,
		Enchant:    item.Enchantments[0], // permanent enchant
		Flags:      uint16(item.ItemFlags),
		Durability: uint16(item.Durability),
		PropertyID: int8(item.RandomPropertiesID),
	}
}

// EntityToItem converts a MongoDB entity to a domain Item.
func EntityToItem(e ItemEntity) *player.Item {
	item := player.NewItem(e.Entry, 0)
	item.StackCount = uint32(e.Count)
	item.ItemFlags = uint32(e.Flags)
	item.Durability = uint32(e.Durability)
	item.RandomPropertiesID = uint32(e.PropertyID)

	if e.Enchant != 0 {
		item.SetEnchant(0, e.Enchant)
	}

	return item
}
