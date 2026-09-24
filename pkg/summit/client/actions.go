package client

import (
	"math"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// Movement flags (pkg/wow/movement.go).
const (
	movementFlagForward = 0x00000001
)

// StandState sends CMSG_STANDSTATECHANGE (0 = stand, 1 = sit, 3 = sleep).
func (wc *WorldClient) StandState(state uint32) {
	pkt := wow.NewPacket(wow.ClientStandstatechange)
	_ = pkt.Write(state)
	wc.Send(pkt)
}

// SetSelection sends CMSG_SET_SELECTION (u64 guid) to target an object.
func (wc *WorldClient) SetSelection(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ClientSetSelection)
	_ = pkt.Write(uint64(guid))
	wc.Send(pkt)
}

// AttackSwing sends CMSG_ATTACKSWING (u64 guid) to start auto-attack. The
// server drives the swing timer from here on.
func (wc *WorldClient) AttackSwing(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ClientAttackswing)
	_ = pkt.Write(uint64(guid))
	wc.Send(pkt)
}

// AttackStop sends CMSG_ATTACKSTOP (empty).
func (wc *WorldClient) AttackStop() {
	wc.Send(wow.NewPacket(wow.ClientAttackstop))
}

// Loot sends CMSG_LOOT (u64 guid) to open a corpse or game object.
func (wc *WorldClient) Loot(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ClientLoot)
	_ = pkt.Write(uint64(guid))
	wc.Send(pkt)
}

// AutostoreLootItem sends CMSG_AUTOSTORE_LOOT_ITEM (u8 slot).
func (wc *WorldClient) AutostoreLootItem(slot uint8) {
	pkt := wow.NewPacket(wow.ClientAutostoreLootItem)
	_ = pkt.WriteOne(int(slot))
	wc.Send(pkt)
}

// LootMoney sends CMSG_LOOT_MONEY (empty).
func (wc *WorldClient) LootMoney() {
	wc.Send(wow.NewPacket(wow.ClientLootMoney))
}

// LootRelease sends CMSG_LOOT_RELEASE (empty).
func (wc *WorldClient) LootRelease() {
	wc.Send(wow.NewPacket(wow.ClientLootRelease))
}

// QueryCreature sends CMSG_CREATURE_QUERY (u32 entry, u64 guid).
func (wc *WorldClient) QueryCreature(entry uint32, guid wow.GUID) {
	pkt := wow.NewPacket(wow.ClientCreatureQuery)
	_ = pkt.Write(entry)
	_ = pkt.Write(uint64(guid))
	wc.Send(pkt)
}

// QueryName sends CMSG_NAME_QUERY (u64 guid).
func (wc *WorldClient) QueryName(guid wow.GUID) {
	pkt := wow.NewPacket(wow.ClientNameQuery)
	_ = pkt.Write(uint64(guid))
	wc.Send(pkt)
}

// SendMovement sends a MSG_MOVE_* packet (u64 guid + MovementInfo). The server
// treats the client-reported position as authoritative.
func (wc *WorldClient) SendMovement(opcode wow.OpCode, flags uint32, pos player.WorldLocation) {
	guid := wc.SelfGUID()

	pkt := wow.NewPacket(opcode)
	_ = pkt.Write(uint64(guid))
	_ = pkt.Write(flags)
	_ = pkt.Write(uint16(0)) // extra flags
	_ = pkt.Write(uint32(time.Now().UnixMilli()))
	_ = pkt.Write(pos.X)
	_ = pkt.Write(pos.Y)
	_ = pkt.Write(pos.Z)
	_ = pkt.Write(pos.O)

	wc.Send(pkt)
}

// OrientationTo returns the WoW orientation (radians, [0, 2π)) pointing from
// one position toward another.
func OrientationTo(from, to player.WorldLocation) float32 {
	o := float32(math.Atan2(float64(to.Y-from.Y), float64(to.X-from.X)))
	if o < 0 {
		o += 2 * math.Pi
	}

	return o
}

// handleAttackStart decodes SMSG_ATTACKSTART (u64 attacker, u64 victim).
func (wc *WorldClient) handleAttackStart(msg *ServerMessage) {
	r := msg.Reader()

	var attacker, victim uint64
	if err := r.Read(&attacker); err != nil {
		return
	}

	if err := r.Read(&victim); err != nil {
		return
	}

	wc.log.Debug().
		Uint64("attacker", attacker).
		Uint64("victim", victim).
		Msg("attack started")
}

// handleAttackStop decodes SMSG_ATTACKSTOP (packed attacker, packed victim, u32).
func (wc *WorldClient) handleAttackStop(msg *ServerMessage) {
	r := msg.Reader()

	_, _ = wow.ReadPackedGUID(r)
	_, _ = wow.ReadPackedGUID(r)

	var unk uint32
	_ = r.Read(&unk)

	wc.log.Debug().Msg("attack stopped")
}

// handleAttackerStateUpdate decodes SMSG_ATTACKERSTATEUPDATE. When the bot is
// the attacker it accumulates the damage on the victim so the AI can infer the
// creature's death (the server never broadcasts NPC health).
func (wc *WorldClient) handleAttackerStateUpdate(msg *ServerMessage) {
	r := msg.Reader()

	var hitInfo uint32
	if err := r.Read(&hitInfo); err != nil {
		return
	}

	attacker, err := wow.ReadPackedGUID(r)
	if err != nil {
		return
	}

	victim, err := wow.ReadPackedGUID(r)
	if err != nil {
		return
	}

	var damage, overkill uint32
	_ = r.Read(&damage)
	_ = r.Read(&overkill)

	var subCount uint8
	_ = r.Read(&subCount)

	for i := 0; i < int(subCount); i++ {
		var (
			school uint32
			dmgF   float32
			dmgI   uint32
		)
		_ = r.Read(&school)
		_ = r.Read(&dmgF)
		_ = r.Read(&dmgI)

		if hitInfo&(hitInfoFullAbsorb|hitInfoPartialAbsorb) != 0 {
			var absorb uint32
			_ = r.Read(&absorb)
		}

		if hitInfo&(hitInfoFullResist|hitInfoPartialResist) != 0 {
			var resist uint32
			_ = r.Read(&resist)
		}
	}

	miss := hitInfo&hitInfoMiss != 0
	crit := hitInfo&hitInfoCritical != 0

	if attacker == wc.SelfGUID() && !miss && damage > 0 {
		wc.addDamageDealt(victim, damage)
	}

	wc.log.Debug().
		Uint64("attacker", uint64(attacker)).
		Uint64("victim", uint64(victim)).
		Uint32("damage", damage).
		Bool("miss", miss).
		Bool("crit", crit).
		Msg("melee hit")
}

// handleXpGain decodes SMSG_LOG_XPGAIN (u64 victim, u32 amount, ...).
func (wc *WorldClient) handleXpGain(msg *ServerMessage) {
	r := msg.Reader()

	var victim uint64
	_ = r.Read(&victim)

	var amount uint32
	_ = r.Read(&amount)

	wc.log.Info().Uint32("amount", amount).Msg("experience gained")
}

// handleLevelUp decodes SMSG_LEVELUP_INFO (u32 level, ...).
func (wc *WorldClient) handleLevelUp(msg *ServerMessage) {
	r := msg.Reader()

	var level uint32
	_ = r.Read(&level)

	wc.log.Info().Uint32("level", level).Msg("level up")
}
