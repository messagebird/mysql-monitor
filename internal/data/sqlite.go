package data

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose"
	"github.com/sirupsen/logrus"
	_ "github.com/messagebird/mysql-monitor/internal/migrations"
	"sync"
)

var defaultDb *DB

var once sync.Once

type DB struct {
	*sqlx.DB

	lock sync.RWMutex

	SlaveStatusService      SlaveStatusService
	MysqlProcessListService MysqlProcessListService
	EngineInnoDBService     EngineInnoDBService
	CycleService            CycleService
	ThreadsService          ThreadsService
	UnixProcessListService  UnixProcessListService
	TopService              TopService
	FileOnDiskService       FileOnDiskService
}

type l struct {

}

func (l) Lock() {

}
func (l) Unlock() {
}

func (l) RLock() {

}
func (l) RUnlock() {
}

func GetDB(file string) *DB {
	once.Do(func() {
		d, err := sqlx.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=true&_ignore_check_constraints=false&_auto_vacuum=2&_sync=3&cache=shared", file))
		if err != nil {
			logrus.WithError(err).Fatal("could not open sqlite3 DB")
		}

		err = d.Ping()
		if err != nil {
			logrus.WithError(err).Fatal("pinning sqlite DB failed")
		}

		d.SetMaxOpenConns(1)

		_ = goose.SetDialect("sqlite3")
		err = goose.Up(d.DB, ".")
		if err != nil {
			logrus.WithError(err).Fatal("could not migrate database")
		}

		defaultDb = &DB{
			DB: d,
		}

		defaultDb.SlaveStatusService = SlaveStatusService{db: defaultDb}
		defaultDb.MysqlProcessListService = MysqlProcessListService{db: defaultDb}
		defaultDb.EngineInnoDBService = EngineInnoDBService{db: defaultDb}
		defaultDb.CycleService = CycleService{db: defaultDb}
		defaultDb.ThreadsService = ThreadsService{db: defaultDb}
		defaultDb.UnixProcessListService = UnixProcessListService{db: defaultDb}
		defaultDb.TopService = TopService{db: defaultDb}
		defaultDb.FileOnDiskService = FileOnDiskService{db: defaultDb}
	})

	return defaultDb
}
