package world

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/basedata"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/summit/world/quest"
	"github.com/paalgyula/summit/pkg/wow"
	"github.com/paalgyula/summit/pkg/wow/protocol"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createQuestTestHarness(t *testing.T) (*WorldSession, *Server, chan *wow.Packet, func()) {
	t.Helper()

	c1, c2 := net.Pipe()
	sock := protocol.NewWoWSocket(c2)

	srv := &Server{
		log:      zerolog.Nop(),
		spawns:   NewSpawnManager(),
		questMgr: quest.NewManager(nil),
	}

	bd := &basedata.Store{}
	basedata.SetInstance(bd)
	srv.baseData = bd

	p := createTestPlayer()
	p.Inventory = player.NewInventory()
	p.Level = 10
	p.Race = wow.PlayerRace(1)   // Human
	p.Class = wow.PlayerClass(1) // Warrior
	p.QuestStatus = make(map[uint32]*quest.QuestStatusData)
	p.RewardedQuests = make(map[uint32]bool)

	ws := &WorldSession{
		ws:     srv,
		socket: sock,
		player: p,
		log:    zerolog.Nop(),
	}

	ch := make(chan *wow.Packet, 50)
	go func() {
		for {
			header := make([]byte, 4)
			if _, err := io.ReadFull(c1, header); err != nil {
				return
			}
			size := binary.BigEndian.Uint16(header[0:2])
			opcode := binary.LittleEndian.Uint16(header[2:4])
			body := make([]byte, size-2)
			if _, err := io.ReadFull(c1, body); err != nil {
				return
			}
			pkt := wow.NewPacket(wow.OpCode(opcode))
			_ = pkt.Write(body)
			ch <- pkt
		}
	}()

	cleanup := func() {
		_ = c1.Close()
		_ = c2.Close()
	}

	return ws, srv, ch, cleanup
}

func readExpectedPacket(t *testing.T, ch chan *wow.Packet, opcode wow.OpCode, timeout time.Duration) *wow.Packet {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case p := <-ch:
			if p.Opcode() == opcode {
				return p
			}
		case <-deadline:
			t.Fatalf("timed out waiting for opcode %v (0x%04x)", opcode, int(opcode))
			return nil
		}
	}
}

func TestQuestFinisher_AutoOpenOfferReward(t *testing.T) {
	ws, srv, ch, cleanup := createQuestTestHarness(t)
	defer cleanup()

	// Setup NPC
	npc := &NPC{
		EntryID:  1234,
		Name:     "Quest Finisher",
		NpcFlags: 2, // Questgiver
	}
	srv.spawns.SpawnNPC(npc)
	npcGUID := wow.NewGUID(wow.UnitGUID, uint32(npc.SpawnID))

	// Setup Quest: completed by NPC 1234
	srv.questMgr.AddQuestTemplate(&quest.Quest{
		ID:          1001,
		Title:       "Test Slaying Quest",
		Description: "You did it!",
		Level:       5,
		RewardXP:    200,
		RewardMoney: 50,
	})
	srv.questMgr.AddQuestInvolved(1234, 1001)

	// Set player quest state to Complete
	pQuests := ws.getPlayerQuests()
	pQuests[1001] = &quest.QuestStatusData{
		Status: quest.QuestStatusComplete,
	}
	ws.addQuestIDToLog(1001)
	ws.setQuestLogState(1001, QuestSlotStateComplete)

	// Player talks to NPC: CMSG_QUESTGIVER_HELLO
	helloPkt := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt.Write(uint64(npcGUID))
	ws.HandleQuestgiverHello(helloPkt.Bytes())

	// Should auto-open offer reward directly (SMSG_QUESTGIVER_OFFER_REWARD)
	resp := readExpectedPacket(t, ch, wow.ServerQuestgiverOfferReward, time.Second)
	require.NotNil(t, resp)

	// Player turns in quest: CMSG_QUESTGIVER_CHOOSE_REWARD
	choosePkt := wow.NewPacket(wow.ClientQuestgiverChooseReward)
	_ = choosePkt.Write(uint64(npcGUID))
	_ = choosePkt.WriteUint32(1001)
	_ = choosePkt.WriteUint32(0) // choice 0
	ws.HandleQuestgiverChooseReward(choosePkt.Bytes())

	// Server sends SMSG_QUESTGIVER_QUEST_COMPLETE
	compResp := readExpectedPacket(t, ch, wow.ServerQuestgiverQuestComplete, time.Second)
	require.NotNil(t, compResp)

	// And SMSG_GOSSIP_COMPLETE
	closeResp := readExpectedPacket(t, ch, wow.ServerGossipComplete, time.Second)
	require.NotNil(t, closeResp)

	// Check player was rewarded
	assert.Equal(t, uint32(200), ws.player.XP)
	assert.Equal(t, uint32(50), ws.player.Money)
	assert.True(t, ws.player.RewardedQuests.(map[uint32]bool)[1001])
}

func TestQuestFinisher_IncompleteShowsRequestItems(t *testing.T) {
	ws, srv, ch, cleanup := createQuestTestHarness(t)
	defer cleanup()

	// Setup NPC
	npc := &NPC{
		EntryID:  1235,
		Name:     "Quest Finisher 2",
		NpcFlags: 2,
	}
	srv.spawns.SpawnNPC(npc)
	npcGUID := wow.NewGUID(wow.UnitGUID, uint32(npc.SpawnID))

	// Setup Quest: completed by NPC 1235
	srv.questMgr.AddQuestTemplate(&quest.Quest{
		ID:                   1002,
		Title:                "Incomplete Quest",
		Level:                5,
		RequiredNpcOrGo:      [4]int32{500, 0, 0, 0},
		RequiredNpcOrGoCount: [4]uint16{5, 0, 0, 0},
	})
	srv.questMgr.AddQuestInvolved(1235, 1002)

	// Player quest state is Incomplete
	pQuests := ws.getPlayerQuests()
	pQuests[1002] = &quest.QuestStatusData{
		Status:            quest.QuestStatusIncomplete,
		CreatureOrGOCount: [4]uint16{2, 0, 0, 0},
	}
	ws.addQuestIDToLog(1002)

	// Player talks to NPC: CMSG_QUESTGIVER_HELLO
	helloPkt := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt.Write(uint64(npcGUID))
	ws.HandleQuestgiverHello(helloPkt.Bytes())

	// Incomplete quest shows SMSG_QUESTGIVER_REQUEST_ITEMS (with canComplete = false)
	resp := readExpectedPacket(t, ch, wow.ServerQuestgiverRequestItems, time.Second)
	require.NotNil(t, resp)
}

func TestQuestAccept_TalkToQuest_ImmediatelyComplete(t *testing.T) {
	ws, srv, ch, cleanup := createQuestTestHarness(t)
	defer cleanup()

	// Setup NPC Willem (starter) and McBride (ender)
	willem := &NPC{EntryID: 823, Name: "Deputy Willem", NpcFlags: 2}
	srv.spawns.SpawnNPC(willem)
	willemGUID := wow.NewGUID(wow.UnitGUID, uint32(willem.SpawnID))

	mcbride := &NPC{EntryID: 197, Name: "Marshal McBride", NpcFlags: 2}
	srv.spawns.SpawnNPC(mcbride)
	mcbrideGUID := wow.NewGUID(wow.UnitGUID, uint32(mcbride.SpawnID))

	// Quest 783: A Threat Within (talk-to quest, 0 required kills, 0 required items)
	srv.questMgr.AddQuestTemplate(&quest.Quest{
		ID:          783,
		Title:       "A Threat Within",
		Description: "Speak with Marshal McBride.",
		Level:       1,
		RewardXP:    100,
		RewardMoney: 25,
	})
	srv.questMgr.AddQuestStarter(823, 783)
	srv.questMgr.AddQuestInvolved(197, 783)

	// Accept Quest 783 from Willem
	acceptPkt := wow.NewPacket(wow.ClientQuestgiverAcceptQuest)
	_ = acceptPkt.Write(uint64(willemGUID))
	_ = acceptPkt.WriteUint32(783)
	ws.HandleQuestgiverAcceptQuest(acceptPkt.Bytes())

	// Should have added quest to log and marked as Complete immediately
	pQuests := ws.getPlayerQuests()
	require.NotNil(t, pQuests[783])
	assert.Equal(t, quest.QuestStatusComplete, pQuests[783].Status)
	assert.Equal(t, uint32(783), ws.questLog[0])

	// Should have sent SMSG_QUESTUPDATE_COMPLETE on accept
	compPkt := readExpectedPacket(t, ch, wow.ServerQuestupdateComplete, time.Second)
	require.NotNil(t, compPkt)

	// Player goes to Marshal McBride: CMSG_QUESTGIVER_HELLO
	helloPkt := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt.Write(uint64(mcbrideGUID))
	ws.HandleQuestgiverHello(helloPkt.Bytes())

	// Should auto-open offer reward directly (SMSG_QUESTGIVER_OFFER_REWARD)
	resp := readExpectedPacket(t, ch, wow.ServerQuestgiverOfferReward, time.Second)
	require.NotNil(t, resp)
}

func TestQuestFinisher_TalkToQuest_IncompleteStatusUpgradedOnHello(t *testing.T) {
	ws, srv, ch, cleanup := createQuestTestHarness(t)
	defer cleanup()

	// Setup McBride
	mcbride := &NPC{EntryID: 197, Name: "Marshal McBride", NpcFlags: 2}
	srv.spawns.SpawnNPC(mcbride)
	mcbrideGUID := wow.NewGUID(wow.UnitGUID, uint32(mcbride.SpawnID))

	srv.questMgr.AddQuestTemplate(&quest.Quest{
		ID:          783,
		Title:       "A Threat Within",
		Description: "Speak with Marshal McBride.",
		Level:       1,
		RewardXP:    100,
	})
	srv.questMgr.AddQuestInvolved(197, 783)

	// Simulate existing player who accepted quest earlier with Incomplete status
	pQuests := ws.getPlayerQuests()
	pQuests[783] = &quest.QuestStatusData{
		Status: quest.QuestStatusIncomplete,
	}
	ws.addQuestIDToLog(783)

	// Player talks to Marshal McBride: CMSG_QUESTGIVER_HELLO
	helloPkt := wow.NewPacket(wow.ClientQuestgiverHello)
	_ = helloPkt.Write(uint64(mcbrideGUID))
	ws.HandleQuestgiverHello(helloPkt.Bytes())

	// Status should be upgraded to Complete and SMSG_QUESTGIVER_OFFER_REWARD sent
	assert.Equal(t, quest.QuestStatusComplete, pQuests[783].Status)

	resp := readExpectedPacket(t, ch, wow.ServerQuestgiverOfferReward, time.Second)
	require.NotNil(t, resp)
}


