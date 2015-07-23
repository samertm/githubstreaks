package db

import (
	"fmt"
	"log"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/samertm/githubstreaks/conf"
)

var driverName = "postgres"

var DB *sqlx.DB

func init() {
	db, err := sqlx.Connect(driverName, conf.Config.PostgresDataSource)
	if err != nil {
		log.Printf("Could not connect to db: %s", err)
	}
	DB = db
}

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

// GetSetMock creates a sqlx.DB pointer created by the sqlmock
// package, sets the global database to that pointer, and returns it.
// It is used for testing. If there is an error creating the mock,
// GetMock panics.
//
// Keep in mind, sqlmock cannot be used in parallel tests.
func GetSetMock() *sqlx.DB {
	db, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	dbx := sqlx.NewDb(db, driverName)
	DB = dbx
	return dbx
}
