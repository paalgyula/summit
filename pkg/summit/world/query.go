package world

import (
	"github.com/paalgyula/summit/pkg/store"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// HandleNameQuery answers CMSG_NAME_QUERY (u64 guid) with SMSG_NAME_QUERY_RESPONSE.
// Online players are resolved from their sessions, everyone else from the store.
func (gc *WorldSession) HandleNameQuery(data wow.PacketData) {
	reader := wow.NewPacketReader(data)

	var rawGUID uint64
	if err := reader.Read(&rawGUID); err != nil {
		return
	}

	guid := wow.GUID(rawGUID)

	var found *player.Player

	if server, ok := gc.ws.(*Server); ok {
		for _, other := range server.GetOnlineSessions() {
			if other.player != nil && other.player.GUID() == guid {
				found = other.player

				break
			}
		}
	}

	if found == nil {
		p, err := gc.ws.GetCharacter(guid.Counter())
		if err == nil && p != nil {
			found = p
		}
	}

	pkt := wow.NewPacket(wow.ServerNameQueryResponse)
	pkt.WriteBytes(guid.Pack())

	if found == nil {
		_ = pkt.WriteOne(1) // name unknown
		gc.socket.Send(pkt)

		return
	}

	_ = pkt.WriteOne(0) // name known
	pkt.WriteString(found.Name)
	_ = pkt.WriteOne(0) // realm name (empty: same realm)
	_ = pkt.WriteOne(int(found.Race))
	_ = pkt.WriteOne(int(found.Gender))
	_ = pkt.WriteOne(int(found.Class))
	_ = pkt.WriteOne(0) // no declined names

	gc.socket.Send(pkt)
}

// HandleCreatureQuery answers CMSG_CREATURE_QUERY (u32 entry, u64 guid) with
// SMSG_CREATURE_QUERY_RESPONSE in the 3.3.5a layout.
func (gc *WorldSession) HandleCreatureQuery(data wow.PacketData) {
	reader := wow.NewPacketReader(data)

	var entry uint32
	if err := reader.Read(&entry); err != nil {
		return
	}

	var tmpl *store.CreatureTemplate

	if server, ok := gc.ws.(*Server); ok && server.spawns != nil {
		tmpl = server.spawns.Template(entry)
	}

	pkt := wow.NewPacket(wow.ServerCreatureQueryResponse)

	if tmpl == nil {
		_ = pkt.Write(entry | 0x80000000) // "no such creature"
		gc.socket.Send(pkt)

		return
	}

	_ = pkt.Write(entry)
	pkt.WriteString(tmpl.Name)
	pkt.WriteString("") // name 2-4 are unused
	pkt.WriteString("")
	pkt.WriteString("")
	pkt.WriteString(tmpl.SubName)
	pkt.WriteString("")        // icon name
	_ = pkt.Write(uint32(0))   // type flags
	_ = pkt.Write(tmpl.Type)   // creature type (beast, humanoid...)
	_ = pkt.Write(tmpl.Family) // creature family
	_ = pkt.Write(tmpl.Rank)   // rank (normal, elite, boss)
	_ = pkt.Write(uint32(0))   // kill credit 1
	_ = pkt.Write(uint32(0))   // kill credit 2

	for i := 0; i < 4; i++ {
		var id uint32
		if i < len(tmpl.ModelIDs) {
			id = tmpl.ModelIDs[i]
		}

		_ = pkt.Write(id)
	}

	_ = pkt.Write(tmpl.HealthMultiplier)
	_ = pkt.Write(float32(1)) // mana multiplier
	_ = pkt.WriteOne(0)       // racial leader

	for i := 0; i < 6; i++ {
		_ = pkt.Write(uint32(0)) // quest items
	}

	_ = pkt.Write(tmpl.MovementType)

	gc.socket.Send(pkt)
}
