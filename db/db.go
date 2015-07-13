package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/samertm/githubstreaks/conf"
)

var DB *sqlx.DB = sqlx.MustConnect("postgres", conf.Config.PostgresDataSource)

type Binder struct {
	Len   int
	Items []interface{}
}

// Returns "$b.Len".
func (b *Binder) Bind(i interface{}) string {
	b.Items = append(b.Items, i)
	b.Len++
	return fmt.Sprintf("$%d", b.Len)
}
