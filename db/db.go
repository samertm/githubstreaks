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
func (b *Binder) Bind(vs ...interface{}) string {
	var str string
	for i, v := range vs {
		b.Items = append(b.Items, v)
		b.Len++
		str += fmt.Sprintf("$%d", b.Len)
		if i < len(vs)-1 {
			str += ", "
		}
	}
	return str
}
