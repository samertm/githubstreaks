package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/samertm/githubstreaks/conf"
)

var DB *sqlx.DB = sqlx.MustConnect("postgres", fmt.Sprintf("sslmode=disable dbname=%s user=%s", conf.Config.PGDATABASE, conf.Config.PGUSER))
