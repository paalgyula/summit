package docs

import (
	"fmt"
	"strconv"
	"time"
)

var Version = "dev"
var Branch = "master"
var Gitsha = "-"
var Compiled = fmt.Sprintf("%d", time.Now().UnixMilli()/1000)
var Buildhost = "localhost"

func BuildInfo() string {
	compileTime := time.Now().Format("2006-01-02 15:04:05")
	if Compiled != "now" {
		ct, err := strconv.ParseInt(Compiled, 10, 64)
		if err == nil {
			compileTime = time.UnixMilli(ct * 1000).Format("2006-01-02 15:04:05")
		}
	}

	return fmt.Sprintf("Version: %s - Built on: %s at %s Branch: %s (%s)",
		Version,
		Buildhost,
		compileTime,
		Branch,
		Gitsha,
	)
}
