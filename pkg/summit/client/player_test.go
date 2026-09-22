package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/wow"
	"github.com/rs/zerolog"
)

func newTestClient() *WorldClient {
	return &WorldClient{ //nolint:exhaustruct
		log:            zerolog.Nop(),
		closed:         make(chan struct{}),
		charEnumCh:     make(chan []*CharEnum, 1),
		loginVerifyCh:  make(chan struct{}, 1),
		loginFailedCh:  make(chan uint8, 1),
		readyCh:        make(chan struct{}, 1),
		clientMessages: make(chan *wow.Packet, 64),
		objects:        make(map[wow.GUID]*Entity),
		creatureNames:  make(map[uint32]string),
		spellFailures:  make(map[uint32]SpellFailure),
	}
}

func TestPickCharacter(t *testing.T) {
	chars := []*CharEnum{{Name: "Thrall"}, {Name: "Jaina"}} //nolint:exhaustruct

	if got := pickCharacter(nil, ""); got != nil {
		t.Fatalf("expected nil for empty list, got %v", got)
	}

	if got := pickCharacter(chars, ""); got == nil || got.Name != "Thrall" {
		t.Fatalf("expected first character, got %v", got)
	}

	if got := pickCharacter(chars, "jaina"); got == nil || got.Name != "Jaina" {
		t.Fatalf("expected case-insensitive match, got %v", got)
	}

	if got := pickCharacter(chars, "missing"); got != nil {
		t.Fatalf("expected nil for missing character, got %v", got)
	}
}

func TestLoginVerifyWorldPopulatesPlayer(t *testing.T) {
	wc := newTestClient()

	var payload bytes.Buffer
	_ = binary.Write(&payload, binary.LittleEndian, uint32(1))    // map
	_ = binary.Write(&payload, binary.LittleEndian, float32(1.5)) // x
	_ = binary.Write(&payload, binary.LittleEndian, float32(2.5)) // y
	_ = binary.Write(&payload, binary.LittleEndian, float32(3.5)) // z
	_ = binary.Write(&payload, binary.LittleEndian, float32(0.5)) // o

	wc.setSelf(&Player{Name: "Thrall"})
	wc.handleLoginVerifyWorld(&ServerMessage{Opcode: wow.ServerLoginVerifyWorld, Data: payload.Bytes()})

	select {
	case <-wc.loginVerifyCh:
	default:
		t.Fatal("expected login verify signal")
	}

	p := wc.Player()
	if p == nil || !p.InWorld {
		t.Fatalf("expected player in world, got %+v", p)
	}

	if p.Location.Map != 1 || p.Location.X != 1.5 || p.Location.Y != 2.5 || p.Location.Z != 3.5 {
		t.Fatalf("unexpected location: %+v", p.Location)
	}
}

func TestInitialStateBecomesReady(t *testing.T) {
	wc := newTestClient()
	wc.setSelf(&Player{Name: "Thrall"})

	var verify bytes.Buffer
	_ = binary.Write(&verify, binary.LittleEndian, uint32(0))
	_ = binary.Write(&verify, binary.LittleEndian, float32(0))
	_ = binary.Write(&verify, binary.LittleEndian, float32(0))
	_ = binary.Write(&verify, binary.LittleEndian, float32(0))
	_ = binary.Write(&verify, binary.LittleEndian, float32(0))
	wc.handleLoginVerifyWorld(&ServerMessage{Opcode: wow.ServerLoginVerifyWorld, Data: verify.Bytes()})

	// SMSG_INITIAL_SPELLS: u8 spec, u16 count, (u32 spell, u16 unk) x count, u16 cooldowns
	var spells bytes.Buffer
	_ = spells.WriteByte(0)
	_ = binary.Write(&spells, binary.LittleEndian, uint16(2))
	_ = binary.Write(&spells, binary.LittleEndian, uint32(133))
	_ = binary.Write(&spells, binary.LittleEndian, uint16(0))
	_ = binary.Write(&spells, binary.LittleEndian, uint32(116))
	_ = binary.Write(&spells, binary.LittleEndian, uint16(0))
	_ = binary.Write(&spells, binary.LittleEndian, uint16(0))
	wc.handleInitialSpells(&ServerMessage{Opcode: wow.ServerInitialSpells, Data: spells.Bytes()})

	// SMSG_ACTION_BUTTONS: u8 kind=1, then 144 u32 buttons
	var actions bytes.Buffer
	_ = actions.WriteByte(1)
	for i := 0; i < maxActionButtons; i++ {
		_ = binary.Write(&actions, binary.LittleEndian, uint32(i+1))
	}
	wc.handleActionButtons(&ServerMessage{Opcode: wow.ServerActionButtons, Data: actions.Bytes()})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	p, err := wc.WaitForInitialState(ctx)
	if err != nil {
		t.Fatalf("expected ready state, got %v", err)
	}

	if len(p.KnownSpells) != 2 || p.KnownSpells[0] != 133 || p.KnownSpells[1] != 116 {
		t.Fatalf("unexpected spells: %v", p.KnownSpells)
	}

	if len(p.Actions) != maxActionButtons || p.Actions[0] != 1 {
		t.Fatalf("unexpected actions: %v", p.Actions)
	}
}

func TestCharacterLoginFailedSignals(t *testing.T) {
	wc := newTestClient()
	wc.handleCharacterLoginFailed(&ServerMessage{
		Opcode: wow.ServerCharacterLoginFailed,
		Data:   []byte{0x2F},
	})

	select {
	case code := <-wc.loginFailedCh:
		if code != 0x2F {
			t.Fatalf("unexpected code: 0x%02x", code)
		}
	default:
		t.Fatal("expected login failed signal")
	}
}
