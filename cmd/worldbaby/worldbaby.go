//nolint:all
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/wow"
)

func main() {
	client, err := babysocket.NewClient()
	if err != nil {
		panic(err)
	}

	defer client.Close()

	client.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("worldbaby connected to babysocket")
	fmt.Println("press CTRL+C to exit")

	<-sigCh

	fmt.Println("\nexiting")
}

func sendVerifyWorld(c *babysocket.Client) {
	p := wow.NewPacket(0x236) // SMSG_LOGIN_VERIFY_WORLD
	p.Write(uint32(1))
	p.Write(float32(10311.3))
	p.Write(float32(832.463))
	p.Write(float32(1326.41))
	p.Write(float32(0.0))

	c.SendToAll(p.OpCode(), p.Bytes())
}

func sendCantLogin(c *babysocket.Client) {
	p := wow.NewPacket(0x041) // SMSG_CHARACTER_LOGIN_FAILED
	p.WriteOne(7)             // LoginFailureReasonLockedByBilling

	c.SendToAll(p.OpCode(), p.Bytes())
}

func enterWorld(c *babysocket.Client) {
	// SMSG_LOGIN_VERIFY_WORLD
	p := wow.NewPacket(0x236)
	p.Write(uint32(580))
	p.Write(float32(10311.3))
	p.Write(float32(832.463))
	p.Write(float32(1326.41))
	p.Write(float32(0.0))
	c.SendToAll(p.OpCode(), p.Bytes())

	// SMSG_FEATURE_SYSTEM_STATUS
	p = wow.NewPacket(0x3C9)
	p.WriteOne(2)
	p.WriteOne(1)
	c.SendToAll(p.OpCode(), p.Bytes())

	// SMSG_TRIGGER_CINEMATIC
	p = wow.NewPacket(wow.ServerTriggerCinematic)
	p.Write(uint32(11))
	c.SendToAll(p.OpCode(), p.Bytes())

	// SMSG_MOTD
	p = wow.NewPacket(0x33D)
	s := "Welcome to Summit!"
	p.Write(uint32(len(s)))
	p.WriteString(s)
	c.SendToAll(p.OpCode(), p.Bytes())

	// SMSG_TUTORIAL_FLAGS
	p = wow.NewPacket(wow.ServerTutorialFlags)
	for i := 0; i < 8; i++ {
		p.Write(uint32(0xFFFFFFFF))
	}
	c.SendToAll(p.OpCode(), p.Bytes())

	// SMSG_LEARNED_DANCE_MOVES
	p = wow.NewPacket(0x455)
	p.Write(uint32(0))
	p.Write(uint32(0))
	c.SendToAll(p.OpCode(), p.Bytes())
}
