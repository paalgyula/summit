//nolint:gochecknoglobals
package docs

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func init() {
	if Buildhost == "-" {
		Buildhost, _ = os.Hostname()
	}
}

var (
	Version   = "dev"
	Branch    = "development"
	Gitsha    = ""
	Compiled  = strconv.FormatInt(time.Now().UnixMilli()/1000, 10)
	Buildhost = "-"
)

func BuildInfo() string {
	compileTime := time.Now().Format("2006-01-02 15:04:05")

	if Compiled != "now" {
		ct, err := strconv.ParseInt(Compiled, 10, 64)
		if err == nil {
			compileTime = time.UnixMilli(ct * 1000).Format("2006-01-02 15:04:05")
		}
	}

	sha := ""
	if Gitsha != "" {
		sha = fmt.Sprintf(" sha(%s)", Gitsha)
	}

	return fmt.Sprintf("Version: %s - Built on: %s at %s Branch: %s%s",
		Version,
		Buildhost,
		compileTime,
		Branch,
		sha,
	)
}
