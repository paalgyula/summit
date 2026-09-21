package mapmanager

import (
	"testing"
	"time"

	"github.com/paalgyula/summit/pkg/summit/world/object"
	"github.com/paalgyula/summit/pkg/summit/world/object/player"
	"github.com/paalgyula/summit/pkg/wow"
)

type countingSender struct{ values int }

// Send counts UPDATETYPE_VALUES packets (block type byte right after the u32 block count).
func (c *countingSender) Send(pkt *wow.Packet) {
	if b := pkt.Bytes(); pkt.Opcode() == wow.ServerUpdateObject && len(b) > 4 && b[4] == wow.UpdateTypeValues {
		c.values++
	}
}

// A changed field must reach every player on the map on the next tick, and
// Update must not deadlock on the objects it drains (ClearUpdateMask used to
// re-enter the map lock).
func TestUpdateDeliversValueChanges(t *testing.T) {
	m := NewMap(1, 0, nil)

	add := func(id uint32) (*player.Player, *countingSender) {
		p := player.NewPlayer()
		p.ID = id
		p.Race, p.Class = wow.RaceOrc, wow.ClassWarior
		p.Init()

		s := &countingSender{}
		p.Sender = s
		m.AddPlayer(p)

		return p, s
	}

	p1, s1 := add(1)
	_, s2 := add(2)

	p1.Object.SetUInt32Value(object.UnitNpcEmotestate, 10)

	done := make(chan struct{})

	go func() {
		m.Update(50)
		m.Update(50)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Map.Update deadlocked")
	}

	if s1.values != 1 || s2.values != 1 {
		t.Fatalf("expected one values update per player, got self=%d other=%d", s1.values, s2.values)
	}

	p1.Object.SetUInt32Value(object.UnitNpcEmotestate, 0)
	m.Update(50)

	if s2.values != 2 {
		t.Fatalf("second change not delivered: other=%d", s2.values)
	}
}
