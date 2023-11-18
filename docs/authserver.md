# 🏔 Auth Server

This is the authentication/realmlist provider component of the Summit WoW emulator.

The current architecture is the following:

![Authentication architecture](auth-server-arch.png)

### Compontents:
- Realmlist provider
- Accounts provider
- gRPC connector (functions for world server)

All components are pluggable, you can write your own implementation if you like to


# Running an auth server

There are different options:
- From binary distribution, downloadable from releases page
- Container - (kubernetes deployment/docs later 
- From code:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/paalgyula/summit/docs"
	"github.com/paalgyula/summit/pkg/summit/auth"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
    // Initialize pretty output for the global logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

    // Create a server.
    // This will opens the listener immediately, and 
    // throws error if can't listen on the specified address
	server, err := auth.NewServer("0.0.0.0:5000", &auth.StaticRealmProvider{
		RealmList: []*auth.Realm{
			{
				Icon:          6,
				Lock:          0,
				Flags:         auth.RealmFlagRecommended,
				Name:          "The Highest Summit",
				Address:       "127.0.0.1:5002",
				Population:    3,
				NumCharacters: 1,
				Timezone:      8,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	defer server.Close()


    // because the listener running on a separate goroutine, we are waiting for signals to interrupt it (Interrupt, or Terminate signals)
	done := make(chan bool, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Info().Msg(sig)
		done <- true
	}()

	<-done

	log.Info().Msg("Shutting down")
}
```

## Configuration

@paalgyula - todo