package world

import (
	"math"

	"github.com/paalgyula/summit/pkg/wow"
)

// Graveyard represents a spirit healer / graveyard location.
type Graveyard struct {
	MapID      uint32
	X, Y, Z, O float32
}

// defaultGraveyards holds default starting zone graveyards.
var defaultGraveyards = []Graveyard{
	{MapID: 0, X: -8949.95, Y: -132.66, Z: 83.53, O: 0},   // Northshire / Elwynn
	{MapID: 1, X: -618.5, Y: -4251.6, Z: 38.7, O: 0},     // Valley of Trials / Durotar
	{MapID: 0, X: -6240.0, Y: 331.0, Z: 382.0, O: 0},     // Coldridge / Dun Morogh
	{MapID: 1, X: 10311.0, Y: 832.0, Z: 1326.0, O: 0},    // Shadowglen / Teldrassil
	{MapID: 1, X: -2917.0, Y: -257.0, Z: 53.0, O: 0},     // Red Cloud / Mulgore
	{MapID: 0, X: 1666.0, Y: 1677.0, Z: 120.0, O: 0},     // Deathknell / Tirisfal
	{MapID: 530, X: 10349.0, Y: -6370.0, Z: 33.4, O: 0},  // Sunstrider / Eversong
	{MapID: 530, X: -3962.0, Y: -13931.0, Z: 100.0, O: 0}, // Ammen Vale / Azuremyst
}

// GetClosestGraveyard finds the nearest graveyard to the given coordinates.
func GetClosestGraveyard(mapID uint32, x, y, z float32) Graveyard {
	var best Graveyard
	minDist := float32(math.MaxFloat32)
	found := false

	for _, gy := range defaultGraveyards {
		if gy.MapID != mapID {
			continue
		}
		dx := gy.X - x
		dy := gy.Y - y
		dz := gy.Z - z
		dist := dx*dx + dy*dy + dz*dz
		if dist < minDist {
			minDist = dist
			best = gy
			found = true
		}
	}

	if !found {
		// Fallback to first graveyard or player's bind location
		if len(defaultGraveyards) > 0 {
			return defaultGraveyards[0]
		}
		return Graveyard{MapID: mapID, X: x, Y: y, Z: z, O: 0}
	}

	return best
}

// HandleRepopRequest handles CMSG_REPOP_REQUEST — player clicks "Release Spirit".
func (gc *WorldSession) HandleRepopRequest(data wow.PacketData) {
	if gc.player == nil || gc.player.IsAlive() {
		return
	}

	gc.log.Debug().Msg("handling CMSG_REPOP_REQUEST")

	// Save corpse location
	gc.player.CorpseLocation = gc.player.Location
	gc.player.IsGhost = true
	gc.player.CharFlags |= 0x10 // Dead flag
	gc.player.PlayerFlags |= 0x00000010

	// Find nearest graveyard
	gy := GetClosestGraveyard(gc.player.Location.Map, gc.player.Location.X, gc.player.Location.Y, gc.player.Location.Z)

	// Teleport ghost to graveyard
	gc.TeleportTo(gy.MapID, gy.X, gy.Y, gy.Z, gy.O)

	// Send corpse reclaim delay (0ms)
	pkt := wow.NewPacket(wow.ServerCorpseReclaimDelay)
	_ = pkt.Write(uint32(0))
	gc.socket.Send(pkt)
}

// HandleReclaimCorpse handles CMSG_RECLAIM_CORPSE — player resurrects at their corpse.
func (gc *WorldSession) HandleReclaimCorpse(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsGhost {
		return
	}

	reader := wow.NewPacketReader(data)
	var corpseGUID uint64
	_ = reader.Read(&corpseGUID)

	// Validate distance to corpse
	dx := gc.player.Location.X - gc.player.CorpseLocation.X
	dy := gc.player.Location.Y - gc.player.CorpseLocation.Y
	dz := gc.player.Location.Z - gc.player.CorpseLocation.Z
	distSq := dx*dx + dy*dy + dz*dz

	// Must be within 35 yards of corpse
	if gc.player.Location.Map == gc.player.CorpseLocation.Map && distSq > 35.0*35.0 {
		gc.log.Warn().Float32("distSq", distSq).Msg("player too far from corpse to reclaim")
		return
	}

	gc.log.Info().Msg("player reclaimed corpse and resurrected")

	// Resurrect with 50% HP and 50% Mana
	gc.player.Resurrect(0.5, 0.5)

	// Resend inventory and values update
	gc.sendInventoryUpdate()
}

// HandleResurrectResponse handles CMSG_RESURRECT_RESPONSE.
func (gc *WorldSession) HandleResurrectResponse(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsGhost {
		return
	}

	reader := wow.NewPacketReader(data)
	var targetGUID uint64
	var response uint8
	_ = reader.Read(&targetGUID)
	_ = reader.Read(&response)

	if response == 0 {
		return // Declined
	}

	gc.log.Info().Msg("player accepted resurrection")
	gc.player.Resurrect(0.5, 0.5)
	gc.sendInventoryUpdate()
}

// HandleSpiritHealerActivate handles CMSG_SPIRIT_HEALER_ACTIVATE — resurrect at Spirit Healer.
func (gc *WorldSession) HandleSpiritHealerActivate(data wow.PacketData) {
	if gc.player == nil || !gc.player.IsGhost {
		return
	}

	gc.log.Info().Msg("player resurrected via spirit healer")

	// Resurrect with 25% HP and Mana
	gc.player.Resurrect(0.25, 0.25)
	gc.sendInventoryUpdate()
}
