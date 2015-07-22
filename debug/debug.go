package debug

import (
	"github.com/samertm/githubstreaks/conf"
	"github.com/tj/go-debug"
)

var Logf = debug.Debug("single")

func init() {
	if conf.Config.Debug != "" {
		debug.Enable(conf.Config.Debug)
	}
}
