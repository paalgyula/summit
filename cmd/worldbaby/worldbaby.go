package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/pkg/summit/world/babysocket"
	"github.com/paalgyula/summit/pkg/summit/world/packets"
)

func main() {
	c, err := babysocket.NewClient()
	if err != nil {
		panic(err)
	}

	c.Start()

	done := make(chan bool, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	c.SendToAll(packets.ServerPong.Int(), make([]byte, 4))

	fmt.Println("awaiting interrupt signal (CTRL+C)")
	<-done
	fmt.Println("exiting")
}
