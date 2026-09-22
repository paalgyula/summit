package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

// ErrNoCharacters is returned by EnterWorld when the account has no characters.
var ErrNoCharacters = errors.New("account has no characters")

// errConnectionClosed is returned when the world connection drops while
// waiting for a server response.
var errConnectionClosed = errors.New("world connection closed")

// maxActionButtons matches the size of the 3.3.5a action bar packet
// (world.MaxActionButtons).
const maxActionButtons = 144

// Player is the bot's own character once it has entered the world. It is filled
// from the character list and the initial world packets.
type Player struct {
	GUID   wow.GUID
	Name   string
	Race   wow.PlayerRace
	Class  wow.PlayerClass
	Gender wow.PlayerGender
	Level  uint8

	Location     player.WorldLocation
	BindLocation player.WorldLocation

	KnownSpells []uint32
	Actions     []uint32

	// InWorld is true once SMSG_LOGIN_VERIFY_WORLD has been received.
	InWorld bool

	knownSpellsReceived bool
	actionsReceived     bool
}

// setCharacters stores the latest character list and notifies any waiter.
func (wc *WorldClient) setCharacters(chars []*CharEnum) {
	wc.stateMu.Lock()
	wc.characters = chars
	wc.stateMu.Unlock()

	select {
	case wc.charEnumCh <- chars:
	default:
	}
}

// Characters returns the most recent character list, or nil when none arrived.
func (wc *WorldClient) Characters() []*CharEnum {
	wc.stateMu.RLock()
	defer wc.stateMu.RUnlock()

	return wc.characters
}

// Player returns a snapshot of the bot's own player once it has entered the
// world, or nil before that.
func (wc *WorldClient) Player() *Player {
	wc.stateMu.RLock()
	defer wc.stateMu.RUnlock()

	if wc.self == nil {
		return nil
	}

	snapshot := *wc.self

	return &snapshot
}

// setSelf initializes the player state for a character login request.
func (wc *WorldClient) setSelf(p *Player) {
	wc.stateMu.Lock()
	wc.self = p
	wc.stateMu.Unlock()
}

// setSelfGUID records the bot's own GUID in the object manager.
func (wc *WorldClient) setSelfGUID(guid wow.GUID) {
	wc.objectsMu.Lock()
	wc.selfGUID = guid
	wc.objectsMu.Unlock()
}

// applyPlayer mutates the current player state under the state lock.
func (wc *WorldClient) applyPlayer(mutate func(*Player)) {
	wc.stateMu.Lock()
	if wc.self != nil {
		mutate(wc.self)
	}
	wc.stateMu.Unlock()
}

// maybeReady signals readyCh once the player is in the world with its initial
// spellbook and action bar populated.
func (wc *WorldClient) maybeReady() {
	wc.stateMu.RLock()
	p := wc.self
	ready := p != nil && p.InWorld && p.knownSpellsReceived && p.actionsReceived
	wc.stateMu.RUnlock()

	if ready {
		select {
		case wc.readyCh <- struct{}{}:
		default:
		}
	}
}

// WaitForCharacters blocks until the server sends a character list, or the
// context/connection ends.
func (wc *WorldClient) WaitForCharacters(ctx context.Context) ([]*CharEnum, error) {
	if chars := wc.Characters(); len(chars) > 0 {
		return chars, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wc.closed:
		return nil, errConnectionClosed
	case <-wc.charEnumCh:
		return wc.Characters(), nil
	}
}

// EnterWorld selects a character and drives CMSG_PLAYER_LOGIN, blocking until
// the server confirms the spawn (SMSG_LOGIN_VERIFY_WORLD) or the context ends.
//
// selector matches a character name case-insensitively; an empty selector picks
// the first character on the account.
func (wc *WorldClient) EnterWorld(ctx context.Context, selector string) (*Player, error) {
	chars, err := wc.WaitForCharacters(ctx)
	if err != nil {
		return nil, err
	}

	char := pickCharacter(chars, selector)
	if char == nil {
		if selector == "" {
			return nil, ErrNoCharacters
		}

		return nil, fmt.Errorf("no character named %q", selector)
	}

	wc.drainLoginSignals()

	wc.setSelf(&Player{
		GUID:     char.GUID,
		Name:     char.Name,
		Race:     char.Race,
		Class:    char.Class,
		Gender:   char.Gender,
		Level:    char.Level,
		Location: char.Location,
	})
	wc.setSelfGUID(char.GUID)

	pkt := wow.NewPacket(wow.ClientPlayerLogin)
	if err := pkt.Write(uint64(char.GUID)); err != nil {
		return nil, fmt.Errorf("write player login: %w", err)
	}

	wc.Send(pkt)
	wc.log.Info().Str("name", char.Name).Msg("requesting character login")

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wc.closed:
		return nil, errConnectionClosed
	case code := <-wc.loginFailedCh:
		return nil, fmt.Errorf("character login failed (0x%02x)", code)
	case <-wc.loginVerifyCh:
		return wc.Player(), nil
	}
}

// WaitForInitialState blocks until the login sequence completed: the player is
// in the world and the initial spellbook and action bar have been received.
func (wc *WorldClient) WaitForInitialState(ctx context.Context) (*Player, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wc.closed:
		return nil, errConnectionClosed
	case <-wc.readyCh:
		return wc.Player(), nil
	}
}

// drainLoginSignals clears any stale login result from a previous attempt.
func (wc *WorldClient) drainLoginSignals() {
	for {
		select {
		case <-wc.loginVerifyCh:
		case <-wc.loginFailedCh:
		default:
			return
		}
	}
}

func pickCharacter(chars []*CharEnum, selector string) *CharEnum {
	if len(chars) == 0 {
		return nil
	}

	if selector == "" {
		return chars[0]
	}

	for _, c := range chars {
		if strings.EqualFold(c.Name, selector) {
			return c
		}
	}

	return nil
}

// handleLoginVerifyWorld handles SMSG_LOGIN_VERIFY_WORLD: the server placed the
// character in the world and reports its position.
func (wc *WorldClient) handleLoginVerifyWorld(msg *ServerMessage) {
	r := msg.Reader()

	var loc player.WorldLocation
	if err := r.Read(&loc.Map); err != nil {
		wc.log.Error().Err(err).Msg("cannot read login verify world")
		return
	}

	_ = r.Read(&loc.X)
	_ = r.Read(&loc.Y)
	_ = r.Read(&loc.Z)
	_ = r.Read(&loc.O)

	wc.applyPlayer(func(p *Player) {
		p.Location = loc
		p.InWorld = true
	})

	wc.log.Info().
		Uint32("map", loc.Map).
		Float32("x", loc.X).
		Float32("y", loc.Y).
		Float32("z", loc.Z).
		Float32("o", loc.O).
		Msg("entered world")

	select {
	case wc.loginVerifyCh <- struct{}{}:
	default:
	}
}

// handleCharacterLoginFailed handles SMSG_CHARACTER_LOGIN_FAILED.
func (wc *WorldClient) handleCharacterLoginFailed(msg *ServerMessage) {
	r := msg.Reader()

	var code uint8
	_ = r.Read(&code)

	wc.log.Error().Uint8("code", code).Msg("character login failed")

	select {
	case wc.loginFailedCh <- code:
	default:
	}
}

// handleInitialSpells handles SMSG_INITIAL_SPELLS and stores the spellbook.
func (wc *WorldClient) handleInitialSpells(msg *ServerMessage) {
	r := msg.Reader()

	var spec uint8
	_ = r.Read(&spec)

	var count uint16
	if err := r.Read(&count); err != nil {
		wc.log.Error().Err(err).Msg("cannot read initial spell count")
		return
	}

	spells := make([]uint32, 0, count)

	for i := 0; i < int(count); i++ {
		var (
			spellID uint32
			unk     uint16
		)

		if err := r.Read(&spellID); err != nil {
			break
		}

		_ = r.Read(&unk)
		spells = append(spells, spellID)
	}

	wc.applyPlayer(func(p *Player) {
		p.KnownSpells = spells
		p.knownSpellsReceived = true
	})

	wc.log.Debug().Int("spells", len(spells)).Msg("received initial spells")
	wc.maybeReady()
}

// handleActionButtons handles SMSG_ACTION_BUTTONS and stores the action bar.
func (wc *WorldClient) handleActionButtons(msg *ServerMessage) {
	r := msg.Reader()

	var kind uint8
	if err := r.Read(&kind); err != nil {
		return
	}

	// Only the initial packet carries the full bar; other variants are updates.
	if kind != 1 {
		return
	}

	actions := make([]uint32, 0, maxActionButtons)

	for i := 0; i < maxActionButtons; i++ {
		var button uint32

		if err := r.Read(&button); err != nil {
			break
		}

		actions = append(actions, button)
	}

	wc.applyPlayer(func(p *Player) {
		p.Actions = actions
		p.actionsReceived = true
	})

	wc.log.Debug().Int("actions", len(actions)).Msg("received action bar")
	wc.maybeReady()
}

// handleBindPointUpdate handles SMSG_BINDPOINTUPDATE.
func (wc *WorldClient) handleBindPointUpdate(msg *ServerMessage) {
	r := msg.Reader()

	var loc player.WorldLocation
	_ = r.Read(&loc.X)
	_ = r.Read(&loc.Y)
	_ = r.Read(&loc.Z)
	_ = r.Read(&loc.Map)
	_ = r.Read(&loc.Zone)

	wc.applyPlayer(func(p *Player) { p.BindLocation = loc })

	wc.log.Debug().Uint32("map", loc.Map).Uint32("zone", loc.Zone).Msg("bind point updated")
}

// handleTimeSyncReq handles SMSG_TIME_SYNC_REQ by echoing the counter back with
// the client clock, as the retail client does.
func (wc *WorldClient) handleTimeSyncReq(msg *ServerMessage) {
	r := msg.Reader()

	var counter uint32
	if err := r.Read(&counter); err != nil {
		return
	}

	pkt := wow.NewPacket(wow.ClientTimeSyncResp)
	_ = pkt.Write(counter)
	_ = pkt.Write(uint32(time.Since(wc.startedAt).Milliseconds()))

	wc.Send(pkt)
}
