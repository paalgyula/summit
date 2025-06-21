package world

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"runtime/debug"
	"time"

	"github.com/paalgyula/summit/pkg/wow/protocol"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ServerPacketHeaderSize size of the server's packet header in bytes.
// 2 bytes of length in big endian and 4bytes of opcode in little endian
// byte order.
const ServerPacketHeaderSize = 6

var ErrCannotReadHeader = errors.New("cannot read opcode")

type WorldSession struct {
	ID  string
	n   net.Conn
	log zerolog.Logger

	// Server side generated seed for authentication proofing
	serverSeed []byte

	// crypt *crypt.WowCrypt

	// These is comes from the login (auth) server
	AccountName string
	SessionKey  *big.Int

	ws SessionManager

	// External packet handler connection
	// bs *babysocket.Server

	socket *protocol.WoWSocket

	// KnownCharacterGUIDs stores GUIDs of characters successfully sent in SMSG_CHAR_ENUM
	// Used to validate CMSG_PLAYER_LOGIN requests.
	KnownCharacterGUIDs []wow.GUID
	Player              *player.Player // Active player object for this session

	// Time Sync related fields
	timeSyncCounter         uint32
	pendingTimeSyncRequests map[uint32]uint32 // map[counter]serverTimeSent
	timeSyncClockDelta      int64             // ServerTime - ClientTime (ms)
}

// SetPlayer associates a player object with this session.
func (gc *WorldSession) SetPlayer(p *player.Player) {
	gc.Player = p
}

// GetPlayer retrieves the player object associated with this session.
func (gc *WorldSession) GetPlayer() *player.Player {
	return gc.Player
}

func NewWorldSession(n net.Conn, ws SessionManager, handlers ...PacketHandler) *WorldSession {
	wowsocket := protocol.NewWoWSocket(n)

	//nolint:exhaustruct
	gc := &WorldSession{
		ID: xid.New().String(),
		n:  n,
		log: log.With().
			Caller().
			Str("server", "world").
			Str("addr", n.RemoteAddr().String()).
			Logger(),
		socket: wowsocket,
		ws:     ws,
	}

	// New server seed on connection
	gc.serverSeed = make([]byte, 4)
	_, _ = rand.Read(gc.serverSeed)

	// Register opcode handlers from handlers.go
	gc.RegisterHandlers(handlers...)

	go gc.handleConnection()
	ws.AddClient(gc)

	return gc
}

func (gc *WorldSession) recover() {
	a := recover()
	if a == nil { // No recover needed
		return
	}

	gc.log.Error().Interface("reason", a).Msgf("panic occurred, dropping client")

	r := bufio.NewReader(bytes.NewBuffer(debug.Stack()))
	for i := 0; i < 5; i++ {
		_, _, _ = r.ReadLine()
	}

	stack, _ := io.ReadAll(r)

	fmt.Fprintf(os.Stderr,
		"unhandled client error: \n%s",
		string(stack),
	)

	// Close connection
	gc.Close()
}

// HandlePlayerLogin handles the CMSG_PLAYER_LOGIN opcode.
// This is the point where a player has selected a character and wishes to enter the world.
func (gc *WorldSession) HandlePlayerLogin(pkt *wow.Packet) {
	gc.log.Info().Msg("Received CMSG_PLAYER_LOGIN")

	reader := wow.NewPacketReader(pkt.Bytes()) // Assuming Bytes() gives the raw payload
	var characterGUID wow.GUID
	if err := reader.Read(&characterGUID); err != nil {
		gc.log.Error().Err(err).Msg("Failed to read character GUID from CMSG_PLAYER_LOGIN")
		gc.Close() // Or send an error response
		return
	}

	gc.log.Info().Hex("guid", characterGUID[:]).Msg("Player attempting to login with character")

	// 1. Security Check: Verify characterGUID belongs to gc.AccountName.
	found := false
	for _, knownGUID := range gc.KnownCharacterGUIDs {
		if knownGUID == characterGUID {
			found = true
			break
		}
	}
	if !found {
		gc.log.Warn().Hex("guid", characterGUID[:]).Str("account", gc.AccountName).
			Msg("Player attempted to login with a character GUID not in their known list.")
		// Send ServerCharacterLoginFailed (0x041)
		// loginFailPkt := wow.NewPacket(wow.ServerCharacterLoginFailed)
		// loginFailPkt.WriteOne(wow.CHAR_LOGIN_UNKNOWN_ACCOUNT) // Or another appropriate reason code
		// gc.socket.Send(loginFailPkt)
		gc.Close() // Close connection after sending failure, or client might not process it.
		return
	}
	gc.log.Debug().Hex("guid", characterGUID[:]).Msg("Character GUID is known to this session.")

	// 2. Load Character Data:
	// Assuming gc.ws (SessionManager) has GetCharacterStore() method that returns store.CharacterRepo
	charRepo := gc.ws.GetCharacterStore()
	playerData, err := charRepo.GetCharacterForLogin(characterGUID, gc.AccountName)
	if err != nil {
		gc.log.Error().Err(err).Hex("guid", characterGUID[:]).Str("account", gc.AccountName).
			Msg("Failed to load character data for login")
		loginFailPkt := wow.NewPacket(wow.ServerCharacterLoginFailed)
		// Determine reason based on error, e.g. wow.CHAR_LOGIN_NO_CHARACTER, wow.CHAR_LOGIN_DB_ERROR
		loginFailPkt.WriteOne(byte(wow.CharLoginFailedDbError)) // Using a general DB error for now
		gc.socket.Send(loginFailPkt)
		gc.Close()
		return
	}
	gc.log.Info().Str("charName", playerData.Name).Msg("Character data for login loaded")

	// 3. Create and Populate player.Player Object:
	p := player.NewPlayer() // Assuming NewPlayer() initializes a basic player
	p.ID = playerData.GUID.Counter() // Assuming Player.ID is uint32 and GUID.Counter() gives the low part
	p.Name = playerData.Name
	p.Race = playerData.Race
	p.Class = playerData.Class
	p.Gender = playerData.Gender
	p.Level = playerData.Level // Make sure Player struct has Level
	p.Location = playerData.Location
	// p.SetSession(gc) // If your player object needs a reference back to the session

	gc.SetPlayer(p)
	gc.log.Info().Str("playerName", p.Name).Msg("Player object created and associated with session")

	// 4. Send SMSG_LOGIN_VERIFY_WORLD (0x0236)
	verifyPkt := wow.NewPacket(wow.ServerLoginVerifyWorld)
	verifyPkt.Write(uint32(p.Location.Map)) // MapID
	verifyPkt.Write(p.Location.X)            // X
	verifyPkt.Write(p.Location.Y)            // Y
	verifyPkt.Write(p.Location.Z)            // Z
	verifyPkt.Write(p.Location.O)            // Orientation
	gc.socket.Send(verifyPkt)
	gc.log.Info().Msg("SMSG_LOGIN_VERIFY_WORLD sent")

	// TODO:
	// 5. Send other initial packets (SMSG_ACCOUNT_DATA_TIMES, SMSG_MOTD, SMSG_INITIAL_SPELLS, etc.)
	//    - This will be a sequence of packet creations and sends.

	// TODO:
	// 6. Add Player to Map/World Management System.
	//    - gc.ws.GetMapManager().AddPlayer(p)

	// TODO:
	// 7. Send SMSG_UPDATE_OBJECT (Create Player for Self).
	//    - This is complex and involves building an update mask and field values.

	// TODO:
	// 8. Send SMSG_UPDATE_OBJECT for nearby objects (deferred for now).

	// TODO:
	// 9. Send post-map-add packets (SMSG_TIME_SYNC_REQ, etc.).

	// TODO:
	// 10. Finalize login state (set player as "in world", update DB).

	gc.log.Warn().Msg("CMSG_PLAYER_LOGIN handler is incomplete!")
}

// HandleMovement is the common handler for MSG_MOVE_* opcodes.
func (gc *WorldSession) HandleMovement(pkt *wow.Packet) {
	plr := gc.GetPlayer()
	if plr == nil {
		gc.log.Warn().Str("opcode", pkt.Opcode().String()).Msg("Received movement packet but no player object on session.")
		return
	}

	reader := wow.NewPacketReader(pkt.Bytes())
	var clientGUID wow.GUID
	if err := reader.ReadPackedGUID(&clientGUID); err != nil {
		gc.log.Error().Err(err).Str("opcode", pkt.Opcode().String()).Msg("Failed to read client GUID from movement packet.")
		return
	}

	// Basic validation: Does the GUID in the packet match the player's GUID?
	// More advanced validation would check if player is controlling a vehicle/pet and if clientGUID matches that.
	if clientGUID != plr.GUID() {
		gc.log.Warn().Str("opcode", pkt.Opcode().String()).
			Hex("clientGUID", clientGUID[:]).
			Hex("playerGUID", plr.GUID()[:]).
			Msg("Client sent movement packet with non-matching GUID.")
		return
	}

	moveInfo, err := wow.ReadClientMovementInfo(reader)
	if err != nil {
		gc.log.Error().Err(err).Str("opcode", pkt.Opcode().String()).Msg("Failed to read movement info.")
		return
	}
	moveInfo.UnitGUID = plr.GUID() // Ensure the server's authoritative GUID is set internally

	// Log received movement for debugging
	// gc.log.Debug().Str("opcode", pkt.Opcode().String()).
	// 	Float32("x", moveInfo.X).Float32("y", moveInfo.Y).Float32("z", moveInfo.Z).Float32("o", moveInfo.O).
	// 	Uint32("flags", uint32(moveInfo.Flags)).
	// 	Uint32("time", moveInfo.Timestamp).
	// 	Msg("Received Movement")

	// 	Msg("Received Movement")

	// Adjust client timestamp using clock delta
	adjustedTimestamp := int64(moveInfo.Timestamp) + gc.timeSyncClockDelta
	if gc.timeSyncClockDelta == 0 { // If delta not yet calculated, use client time directly (less accurate)
		adjustedTimestamp = int64(moveInfo.Timestamp)
		// gc.log.Warn().Msg("timeSyncClockDelta is zero, using raw client timestamp for movement.")
	} else if adjustedTimestamp < 0 {
		// This might happen if client time is far ahead or delta is large.
		// Using client timestamp directly might be safer than a negative one.
		// Or, log and investigate. For now, let's cap it or use raw.
		// gc.log.Warn().Int64("adjustedTimestamp", adjustedTimestamp).Msg("Adjusted timestamp is negative, using raw client timestamp.")
		adjustedTimestamp = int64(moveInfo.Timestamp)
	}


	// Update server-side player state
	plr.UpdatePosition(moveInfo.X, moveInfo.Y, moveInfo.Z, moveInfo.O, moveInfo.Pitch)
	plr.SetMovementFlags(moveInfo.Flags)
	// Store the server-aligned timestamp.
	// Note: Timestamps in WoW are complex (tick counts, etc.). This is a simplification.
	plr.SetTimestamp(uint32(adjustedTimestamp))

	// TODO: Handle fall damage if pkt.Opcode() == wow.MsgMoveFallLand
	// This would involve:
	// if pkt.Opcode() == wow.MsgMoveFallLand && moveInfo.HasFallData {
	//    plr.HandleFall(moveInfo.FallTime) // Player needs a HandleFall method
	// }

	// TODO: Handle transport changes if moveInfo.Flags.IsOnTransport() changes state
	// This would involve checking current transport state vs moveInfo.Transport and using a MapManager
	// to add/remove player from transport objects if needed.

	// TODO: Broadcast Movement to Other Clients
	// This requires a MapManager or similar system to find nearby players.
	// broadcastPkt := wow.NewPacket(pkt.Opcode()) // Relay the same opcode
	// wow.WriteServerMovementInfo(broadcastPkt, plr.GetCurrentMovementInfo(), plr.GUID(), pkt.Opcode())
	// gc.ws.GetMapManager().BroadcastToNearbyPlayers(plr, broadcastPkt, false) // false = don't send to self

	if pkt.Opcode() == wow.MsgMoveHeartbeat { // Acknowledge heartbeat to prevent disconnect; can be more sophisticated
		// For now, no explicit ack for heartbeat needed unless client expects one or we do server-side validation against it.
		// TrinityCore sends an empty SMSG_MOVE_UPDATE for heartbeats if no other movement packet is generated.
		// We are relaying the heartbeat (if broadcast is implemented), which is one way to handle it.
	}

	// For now, just log that we're not broadcasting or fully updating state
	gc.log.Debug().Str("opcode", pkt.Opcode().String()).Msg("Movement packet handled (server state update and broadcast are TODO)")
}

func (gc *WorldSession) handleConnection() {
	defer gc.recover() // Panic handler

	time.Sleep(time.Millisecond * 500)
	gc.log.Trace().Msg("sending auth challenge")
	gc.sendAuthChallenge()

	// TODO: Call SendTimeSyncReq() here or after player successfully logs in.
	// For now, let's call it once after auth challenge for testing.
	// A better place might be after SMSG_LOGIN_VERIFY_WORLD or periodically.
	// gc.SendTimeSyncReq() // Call it after HandlePlayerLogin sends initial packets.

	// Handle packets from the channel.
	for pkt := range gc.socket.Packets() {
		gc.Handle(pkt)
	}
}

// SendTimeSyncReq sends an SMSG_TIME_SYNC_REQ to the client.
func (gc *WorldSession) SendTimeSyncReq() {
	if gc.pendingTimeSyncRequests == nil {
		gc.pendingTimeSyncRequests = make(map[uint32]uint32)
	}

	gc.timeSyncCounter++ // Increment for a new request
	counter := gc.timeSyncCounter

	pkt := wow.NewPacket(wow.ServerTimeSyncReq)
	pkt.Write(counter)
	// For 3.3.5a, an additional uint32 (uptime in ms) is sent.
	// This can be 0 if not precisely tracked or a simple uptime.
	pkt.Write(uint32(0)) // Placeholder for server uptime in ms. Actual uptime can be calculated.

	gc.pendingTimeSyncRequests[counter] = uint32(time.Now().UnixMilli())
	gc.socket.Send(pkt)
	// gc.log.Debug().Uint32("counter", counter).Msg("SMSG_TIME_SYNC_REQ sent")
}

// HandleTimeSyncResp handles CMSG_TIME_SYNC_RESP from the client.
func (gc *WorldSession) HandleTimeSyncResp(pkt *wow.Packet) {
	reader := wow.NewPacketReader(pkt.Bytes())
	var clientCounter, clientTimestamp uint32

	if err := reader.Read(&clientCounter); err != nil {
		gc.log.Error().Err(err).Msg("Failed to read clientCounter from CMSG_TIME_SYNC_RESP")
		return
	}
	if err := reader.Read(&clientTimestamp); err != nil {
		gc.log.Error().Err(err).Msg("Failed to read clientTimestamp from CMSG_TIME_SYNC_RESP")
		return
	}

	serverTimeSent, ok := gc.pendingTimeSyncRequests[clientCounter]
	if !ok {
		gc.log.Warn().Uint32("clientCounter", clientCounter).Msg("Received CMSG_TIME_SYNC_RESP for unknown counter.")
		return
	}
	delete(gc.pendingTimeSyncRequests, clientCounter)

	currentTimeMs := uint32(time.Now().UnixMilli())
	roundTripDuration := currentTimeMs - serverTimeSent
	latency := roundTripDuration / 2 // Simplified latency calculation

	// clockDelta = serverTimeAtClientProcessing - clientTimestamp
	// serverTimeAtClientProcessing is approximated as serverTimeSent + latency
	gc.timeSyncClockDelta = int64(serverTimeSent+latency) - int64(clientTimestamp)

	// gc.log.Debug().
	// 	Uint32("clientCounter", clientCounter).
	// 	Uint32("clientTimestamp", clientTimestamp).
	// 	Uint32("serverTimeSent", serverTimeSent).
	// 	Uint32("latency", latency).
	// 	Int64("clockDelta", gc.timeSyncClockDelta).
	// 	Msg("CMSG_TIME_SYNC_RESP processed")
}


func (gc *WorldSession) Close() error {
	gc.ws.Disconnected(gc, "closing GameClient")

	return gc.n.Close() //nolint:wrapcheck
}
