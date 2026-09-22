package client

import (
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// RepopRequest sends CMSG_REPOP_REQUEST ("release spirit").
func (wc *WorldClient) RepopRequest() {
	wc.Send(wow.NewPacket(wow.ClientRepopRequest))
}

// ReclaimCorpse sends CMSG_RECLAIM_CORPSE (u64 corpse guid) to resurrect at the
// corpse. A zero GUID is accepted by the server.
func (wc *WorldClient) ReclaimCorpse(corpse wow.GUID) {
	pkt := wow.NewPacket(wow.ClientReclaimCorpse)
	_ = pkt.Write(uint64(corpse))
	wc.Send(pkt)
}

// SpiritHealerActivate sends CMSG_SPIRIT_HEALER_ACTIVATE (resurrect at the
// spirit healer with 25% health).
func (wc *WorldClient) SpiritHealerActivate() {
	wc.Send(wow.NewPacket(wow.ClientSpiritHealerActivate))
}

// IsDead reports whether the bot's character is currently dead/ghosted.
func (wc *WorldClient) IsDead() bool {
	if self := wc.SelfEntity(); self != nil && self.MaxHealth() > 0 {
		return self.Health() == 0
	}

	return wc.dead.Load()
}

// DeathNoticed is signalled once when the server reports the release location.
func (wc *WorldClient) DeathNoticed() <-chan struct{} {
	return wc.deathCh
}

// ReclaimReady is signalled when the corpse can be reclaimed.
func (wc *WorldClient) ReclaimReady() <-chan struct{} {
	return wc.reclaimCh
}

// DeathLocation returns the last corpse release location.
func (wc *WorldClient) DeathLocation() player.WorldLocation {
	return wc.deathLoc
}

// handleDeathReleaseLoc decodes SMSG_DEATH_RELEASE_LOC (u32 map, f32 x, y, z).
func (wc *WorldClient) handleDeathReleaseLoc(msg *ServerMessage) {
	r := msg.Reader()

	var loc player.WorldLocation

	if err := r.Read(&loc.Map); err != nil {
		return
	}

	_ = r.Read(&loc.X)
	_ = r.Read(&loc.Y)
	_ = r.Read(&loc.Z)

	wc.deathLoc = loc
	wc.dead.Store(true)

	wc.log.Warn().
		Uint32("map", loc.Map).
		Float32("x", loc.X).
		Float32("y", loc.Y).
		Float32("z", loc.Z).
		Msg("player died")

	select {
	case wc.deathCh <- struct{}{}:
	default:
	}
}

// handleCorpseReclaimDelay decodes SMSG_CORPSE_RECLAIM_DELAY (u32 delay ms).
func (wc *WorldClient) handleCorpseReclaimDelay(msg *ServerMessage) {
	r := msg.Reader()

	var delay uint32
	_ = r.Read(&delay)

	wc.log.Debug().Uint32("delay", delay).Msg("corpse reclaim ready")

	select {
	case wc.reclaimCh <- struct{}{}:
	default:
	}
}
