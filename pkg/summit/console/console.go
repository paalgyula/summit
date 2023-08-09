package console

import (
	"bufio"
	"os"
	"strings"

	"github.com/paalgyula/summit/pkg/db"
	"github.com/paalgyula/summit/pkg/summit/world"
	"github.com/rs/zerolog/log"
)

var pWS *world.WorldServer
var pDB *db.Database

func ListenforCommands(ws *world.WorldServer) {
	var s string
	r := bufio.NewReader(os.Stdin)
	pWS = ws
	for {
		s, _ = r.ReadString('\n')
		if s != "" {
			processCommand(strings.TrimSpace(s))
		}
	}
}

func processCommand(cmdstr string) {
	// log.Info().Msg(fmt.Sprintf("Heard command '%s'", cmdstr))

	if pDB == nil {
		pDB = db.GetInstance()
	}

	cmds := strings.Split(cmdstr, " ")
	found := false
	var err error

	cmdlen := len(cmds)

	if cmdlen > 0 {
		if cmds[0] == "account" {
			if cmdlen > 1 {
				if cmds[1] == "create" && cmdlen == 4 {
					if _, err = pDB.CreateAccount(cmds[2], cmds[3]); err != nil {
						log.Err(err).Msg("Account creation failed.")
					} else {
						log.Info().Msg("Account created.")
					}
					found = true
				}

			} else {

			}
		} else if cmds[0] == "server" {
			if cmdlen > 1 {
				if cmds[1] == "stats" {
					pWS.Stats()
					found = true
				}
			}
		}
	}

	if !found {
		log.Info().Msgf("Unknown command: %s", cmdstr)
	}
}
