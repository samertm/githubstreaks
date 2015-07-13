package conf

import (
	"log"

	"github.com/burntsushi/toml"
)

type ConfigVars struct {
	GitHubID           string
	GitHubSecret       string
	BaseURL            string
	PostgresDataSource string
	SessionKey         string
	OAuthStateString   string
}

var Config ConfigVars

func init() {
	if _, err := toml.DecodeFile("conf.toml", &Config); err != nil {
		log.Fatalf("Error decoding conf: %s", err)
	}
}
