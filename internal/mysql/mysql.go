package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-ini/ini"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/messagebird/mysql-monitor/internal/data"
	"github.com/messagebird/mysql-monitor/internal/logging"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"text/template"
	"time"
)

// GetDB returns the database connection that we want to monitor.
// Please make sure that the DB_* environment variables are set.
func GetDB(cnfPath string) (*sqlx.DB, error) {
	dbCred, err := getCredFromCnf(cnfPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not get mysql credentials from cnf file")
	}

	var db *sqlx.DB

	if dbCred.socket == "" {
		db, err = sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/", dbCred.user, dbCred.password, dbCred.host, dbCred.port))
	} else {
		db, err = sqlx.Open("mysql", fmt.Sprintf("%s:%s@unix(%s)/", dbCred.user, dbCred.password, dbCred.socket))
	}

	if err != nil {
		logging.Trace(logging.TraceTypeInside)
		return nil, errors.Wrap(err, "could not open db connection")
	}

	err = db.Ping()

	if err != nil {
		logging.Trace(logging.TraceTypeInside)
		return nil, errors.Wrap(err, "pinging the db failed")
	}

	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(time.Second*2)

	return db, nil
}

type dbCred struct {
	user, password, socket, host, port string
}

func getCredFromCnf(path string) (dbCred, error) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)
	cnf, err := ini.Load(path)
	if err != nil {
		return dbCred{}, errors.Wrap(err, "could not load mysql cnf file")
	}

	cred := dbCred{}

	cred.user = cnf.Section("client").Key("user").String()
	cred.password = cnf.Section("client").Key("password").String()
	cred.socket = cnf.Section("client").Key("socket").String()
	cred.host = cnf.Section("client").Key("host").String()
	cred.port = cnf.Section("client").Key("port").String()

	if cred.user == "" || cred.password == "" {
		logrus.Fatal("cred.user or password is an empty string")
	}

	if cred.host == "" {
		cred.host = "127.0.0.1"
	}

	if cred.port == "" {
		cred.port = "3306"
	}

	return cred, nil
}

type ProcessList struct {
	ID           uint64         `gojay:"id" db:"id"`
	User         string         `gojay:"user" db:"user"`
	Host         string         `gojay:"host" db:"host"`
	DB           sql.NullString `gojay:"db" db:"db"`
	Command      string         `gojay:"command" db:"command"`
	Time         int64          `gojay:"time" db:"time"`
	State        sql.NullString `gojay:"state" db:"state"`
	Info         sql.NullString `gojay:"info" db:"info"`
	RowsSent     uint64         `gojay:"rows_sent" db:"rows_sent"`
	RowsExamined uint64         `gojay:"rows_examined" db:"rows_examined"`
}

// GetProcessList returns a channel of entries pulled from the process list of the passed db.
func GetProcessList(ctx context.Context, db *sqlx.DB) (chan *data.MonitoredData, error) {
	rows, err := db.QueryxContext(ctx, `select id, user, host, db, command, time, state, info from information_schema.PROCESSLIST;`)
	if err != nil {
		return nil, errors.Wrap(err, "could not query db to get process list")
	}

	ch := make(chan *data.MonitoredData)

	go func() {
		defer close(ch)
		defer func() {
			err := rows.Close()
			if err != nil {
				logrus.WithError(err).Error("could not close rows")
			}
		}()

		for rows.Next() {
			var entry data.MysqlProcessList
			err := rows.StructScan(&entry)
			if err != nil {
				logging.Trace(logging.TraceTypeInside)
				logrus.WithError(err).Error("could not scan row for process list")
				continue
			}

			ch <- &data.MonitoredData{MysqlProcessList: &entry}
		}
	}()

	return ch, nil
}

// GetEngineINNODBStatus returns the status of engine INNODB
func GetEngineINNODBStatus(ctx context.Context, db *sqlx.DB) (chan *data.MonitoredData, error) {
	ch := make(chan *data.MonitoredData)

	rows, err := db.QueryxContext(ctx, `SHOW ENGINE INNODB STATUS;`)
	if err != nil {
		return nil, errors.Wrap(err, "could not get engine innodb status")
	}
	go func() {
		defer close(ch)
		defer func() {
			err := rows.Close()
			if err != nil {
				logrus.WithError(err).Error("could not close rows")
			}
		}()

		for rows.Next() {
			r := data.EngineINNODBStatus{}

			err = rows.Scan(&r.Type, &r.Name, &r.Status)
			if err != nil {
				logrus.WithError(err).Error("could not scan row for engine innodb result")
				continue
			}

			ch <- &data.MonitoredData{EngineINNODBStatus: &r}
		}
	}()

	return ch, nil
}

// GetEngineINNODBStatusTemplate returns the template that can be used to parse the result of engine innodb as .txt
func GetEngineINNODBStatusTemplate() (*template.Template, error) {
	t := template.New("Engine innodb status")
	parsed, err := t.Parse(`
======================================================================
		BEGINNING OF ROW
Name: {{ .Name }}
Type: {{ .Type }}
Status: {{ .Status }}
		END OF ROW
======================================================================
	`)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse engine innodb template")
	}

	return parsed, nil
}

type SlaveStatusChan chan *data.SlaveStatus

func (s SlaveStatusChan) IsNil() bool {
	return false
}

// GetSlaveStatus executes `show slave status` and returns it parsed in a struct.
func GetSlaveStatus(ctx context.Context, db *sqlx.DB) (chan *data.MonitoredData, error) {
	rows, err := db.QueryxContext(ctx, `show slave status`)
	if err != nil {
		return nil, errors.Wrap(err, "could not get slave status")
	}

	ch := make(chan *data.MonitoredData)

	go func() {
		logging.Trace(logging.TraceTypeEntering)
		defer logging.Trace(logging.TraceTypeExiting)

		defer close(ch)
		defer func() {
			logging.Trace(logging.TraceTypeEntering)
			defer logging.Trace(logging.TraceTypeExiting)

			if err := rows.Close(); err != nil {
				logrus.WithError(err).Error("could not close rows for slave status")
			}
		}()

		for rows.Next() {
			var status data.SlaveStatus
			err = rows.StructScan(&status)
			if err != nil {
				logrus.WithError(err).Error("could not scan row for show slave status")
				continue
			}

			ch <- &data.MonitoredData{SlaveStatus: &status}
		}
	}()

	return ch, nil
}

// GetThreads runs select * from p_s.threads
func GetThreads(ctx context.Context, db *sqlx.DB) (chan *data.MonitoredData, error) {
	rows, err := db.QueryxContext(ctx,  `
	select thread_id,
       name,
       type,
       processlist_id,
       processlist_user,
       processlist_host,
       processlist_db,
       processlist_command,
       processlist_time,
       processlist_state,
       processlist_info,
       parent_thread_id,
       role,
       instrumented,
       history,
       connection_type,
       thread_os_id
from performance_schema.threads;
	`)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get threads")
	}

	ch := make(chan *data.MonitoredData)

	go func() {
		defer close(ch)
		for rows.Next() {
			var thread data.Thread

			err := rows.StructScan(&thread)
			if err != nil {
				logrus.WithError(err).Error("could not scan row of p_s.threads into struct")
				continue
			}

			ch <- &data.MonitoredData{Thread: &thread}
		}
	}()

	return ch, nil
}
