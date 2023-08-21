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

//nolint:wsl
func main() {
	client, err := babysocket.NewClient()
	if err != nil {
		panic(err)
	}

	client.Start()

	done := make(chan bool, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh

		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("sending to all")

	p := SendCantLogin()
	client.SendToAll(p.OpCode(), p.Bytes())

	// enterWorld(c)

	// c.SendToAll(packets.ServerPong.Int(), make([]byte, 4))

	// fmt.Println("awaiting interrupt signal (CTRL+C)")
	// <-done
	// fmt.Println("exiting")
}

type LoginFailureReason uint8

const (
	LoginFailureReasonFailed             LoginFailureReason = 0
	LoginFailureReasonNoWorld            LoginFailureReason = 1
	LoginFailureReasonDuplicateCharacter LoginFailureReason = 2
	LoginFailureReasonNoInstances        LoginFailureReason = 3
	LoginFailureReasonDisabled           LoginFailureReason = 4
	LoginFailureReasonNoCharacter        LoginFailureReason = 5
	LoginFailureReasonLockedForTransfer  LoginFailureReason = 6
	LoginFailureReasonLockedByBilling    LoginFailureReason = 7
)

func SendVerifyWorld() *wow.Packet {
	p := wow.NewPacket(0x236) // SMSG_LOGIN_VERIFY_WORLD
	p.Write(uint32(1))
	p.Write(float32(10311.3))
	p.Write(float32(832.463))
	p.Write(float32(1326.41))
	p.Write(float32(0.0))

	return p
}

func SendCantLogin() wow.Packet {
	p := wow.NewPacket(0x041) // SMSG_CHARACTER_LOGIN_FAILED
	p.Write(LoginFailureReasonLockedByBilling)

	return *p
}

func enterWorld(c *babysocket.Client) {
	p := wow.NewPacket(0x236) // SMSG_LOGIN_VERIFY_WORLD
	p.Write(uint32(580))
	p.Write(float32(10311.3))
	p.Write(float32(832.463))
	p.Write(float32(1326.41))
	p.Write(float32(0.0))

	c.SendToAll(p.OpCode(), p.Bytes())

	p = wow.NewPacket(0x3C9) // SMSG_FEATURE_SYSTEM_STATUS
	p.WriteOne(2)
	p.WriteOne(1)
	c.SendToAll(p.OpCode(), p.Bytes())

	p = wow.NewPacket(wow.ServerTriggerCinematic)
	p.Write(uint32(11))
	c.SendToAll(p.OpCode(), p.Bytes())

	p = wow.NewPacket(0x33D) // SMSG_MOTD
	s := "Macika"
	p.Write(uint32(len(s)))
	p.WriteString(s)
	c.SendToAll(p.OpCode(), p.Bytes())

	p = wow.NewPacket(wow.ServerTutorialFlags)
	for i := 0; i < 8; i++ {
		p.Write(uint32(0xFFFFFFFF))
	}
	c.SendToAll(p.OpCode(), p.Bytes())

	p = wow.NewPacket(0x455) // SMSG_LEARNED_DANCE_MOVES
	p.Write(uint32(0))
	p.Write(uint32(0))
	c.SendToAll(p.OpCode(), p.Bytes())
}
