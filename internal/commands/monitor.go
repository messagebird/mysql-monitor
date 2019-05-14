package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/messagebird/mysql-monitor/internal/api"
	"github.com/messagebird/mysql-monitor/internal/data"
	"github.com/messagebird/mysql-monitor/internal/gzips"
	"github.com/messagebird/mysql-monitor/internal/logging"
	"github.com/messagebird/mysql-monitor/internal/mysql"
	"github.com/messagebird/mysql-monitor/internal/unix"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func GetMonitorCommandFlags() []cli.Flag {
	return []cli.Flag{
		cli.DurationFlag{
			Name:  "interval, n",
			Usage: "how often should processes be monitored",
			Value: time.Second * 5,
		},
		cli.DurationFlag{
			Name:  "log-retention, r",
			Usage: "how long should the saved data be kept for",
			Value: time.Hour * 24,
		},
		cli.StringFlag{
			Name:  "mysql-credential-path, c",
			Usage: "the path to the mysql cnf file which contains the mysql login credentials",
		},
		cli.StringFlag{
			Name:  "output-dir, o",
			Usage: "the path where the sqlite db/log files containing the data should be saved",
			Value: "./mysql-monitor-db.sqlite",
		},
		cli.BoolFlag{
			Name:  "exclude-mysql-process-list",
			Usage: "if the query 'show full processlist' should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "exclude-innodb-status",
			Usage: "if the query 'SHOW ENGINE INNODB STATUS' should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "exclude-slave-status",
			Usage: "if the query 'SHOW SLAVE STATUS' should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "exclude-ps-threads",
			Usage: "if the query 'select * from performance_schema.threads should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "exclude-unix-process-list",
			Usage: "if the command 'ps aux' should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "exclude-unix-top",
			Usage: "if the command 'top -b -n 1' should not be executed and saved",
		},
		cli.BoolFlag{
			Name:  "save-to-output-dir-file-instead-of-sqlite, not-in-sqlite",
			Usage: "If the logged data should be stored into sqlite which enables this tool to query it with the viewer, or to save raw .txt/.json files to disk. Viewer can not be used if this is enabled.",
		},
		cli.StringFlag{
			Name:  "api-port, p",
			Usage: "On which port the api server should be exposed. This will only be available when the flag --not-in-sqlite is not passed.",
			Value: "9090",
		},
	}
}

func Monitor(cliCtx *cli.Context) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	logrus.Infof("hi, going to run this tool with an interval of %q", cliCtx.Duration("n"))

	abs, err := filepath.Abs(fmt.Sprintf("%s/mysql-monitor.sqlite", cliCtx.String("o")))
	if err != nil {
		logrus.WithError(err).Error("could not determine abs path for output dir")
		return
	}

	dbCredAbs, err := filepath.Abs(cliCtx.String("mysql-credential-path"))
	if err != nil {
		logrus.WithError(err).Error("could not determine abs path for mysql cred")
		return
	}

	monitorDb, err := mysql.GetDB(dbCredAbs)
	if err != nil {
		logrus.WithError(err).Fatal("could not get monitorDb")
	}

	var db *data.DB

	if !cliCtx.Bool("not-in-sqlite") {
		db = data.GetDB(abs)
		defer db.Close()
	}

	var wg sync.WaitGroup

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	exiting := false

	go func() {
		<-c
		exiting = true
		logrus.Info("exiting....")
		cancelFunc()
		wg.Wait()
		logrus.Info("good bye")
		os.Exit(0)
	}()

	go func() {
		ticker := time.Tick(time.Minute)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				wg.Add(1)
				logrus.Debug("running log deletion")
				if cliCtx.Bool("not-in-sqlite") {
					deleteOldCyclesFromDisk(cliCtx.Duration("r"), cliCtx.String("o"))
				} else {
					deleteOldCyclesFromSqlite(cliCtx.Duration("r"), db)
				}
				logrus.Debug("finished log deletion")
				wg.Done()
			}
		}
	}()

	if cliCtx.Bool("not-in-sqlite") {
		runMonitor(cliCtx, monitorDb, db, &wg, ctx)
	} else {
		runMonitorWithViewer(cliCtx, monitorDb, db, &wg, ctx)
	}

	if exiting {
		select {}
	}
}

func runMonitorWithViewer(cliCtx *cli.Context, monitorDb *sqlx.DB, db *data.DB, wg *sync.WaitGroup, ctx context.Context) {
	go runMonitor(cliCtx, monitorDb, db, wg, ctx)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Mount("/api/v1/", api.NewRouter(db))

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", cliCtx.String("p")), r)
	if err != nil {
		logrus.WithError(err).Fatal("could not start api server")
	}
}

func runMonitor(cliCtx *cli.Context, monitorDb *sqlx.DB, db *data.DB, wg *sync.WaitGroup, ctx context.Context) {
	cycles := 0

	ch := make(chan data.CycleData)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				wg.Add(1)
				for e := range ch {
					logrus.Debugf("got chan for %s", e.Path)

					if cliCtx.Bool("not-in-sqlite") {
						err := saveCycleToFile(db, cliCtx.String("o"), e.Path, e.Extension, e.Cycle, e.MonitoredData)
						if err != nil {
							logrus.WithError(err).Error("could not save cycle to file")
						}
					} else {
						saveCycleToDB(db, e.Cycle, e.MonitoredData)
					}
				}
				wg.Done()
				return
			case e := <-ch:
				wg.Add(1)
				logrus.Debugf("got chan for %s", e.Path)

				if cliCtx.Bool("not-in-sqlite") {
					err := saveCycleToFile(db, cliCtx.String("o"), e.Path, e.Extension, e.Cycle, e.MonitoredData)
					if err != nil {
						logrus.WithError(err).Error("could not save cycle to file")
					}
				} else {
					saveCycleToDB(db, e.Cycle, e.MonitoredData)
				}
				wg.Done()
			}
		}
	}()

	time.Sleep(time.Until(time.Now().Round(cliCtx.Duration("n")).Truncate(time.Millisecond)))
	ticker := time.Tick(cliCtx.Duration("n"))

	for {
		select {
		case <-ctx.Done():
			return
		case start := <-ticker:
			cycles++
			wg.Add(1)
			func() {
				defer wg.Done()

				var innerWg sync.WaitGroup
				id := uuid.NewV4()
				cycle := data.Cycle{ID: id, Cycle: cycles, RanAt: start.Round(cliCtx.Duration("n")).Truncate(time.Millisecond)}

				logrus.Infof("running cycle %d", cycles)

				if !cliCtx.Bool("not-in-sqlite") {
					logrus.Debug("saving cycle")
					err := db.CycleService.Save(cycle)
					if err != nil {
						logrus.WithError(err).Fatal("could not save cycle info")
						return
					}
					logrus.Debug("done saving cycle")
				}

				if !cliCtx.Bool("exclude-mysql-process-list") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorMysqlProcessList(monitorDb, &cycle, ch)
					}()
				}

				if !cliCtx.Bool("exclude-innodb-status") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorEngineINNODB(monitorDb, &cycle, ch)
					}()
				}

				if !cliCtx.Bool("exclude-slave-status") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorSlaveStatus(monitorDb, &cycle, ch)
					}()
				}

				if !cliCtx.Bool("exclude-ps-threads") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorThreads(monitorDb, &cycle, ch)
					}()
				}

				if !cliCtx.Bool("exclude-unix-process-list") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorUnixProcessList(&cycle, ch)
					}()
				}

				if !cliCtx.Bool("exclude-unix-top") {
					innerWg.Add(1)
					go func() {
						defer innerWg.Done()
						monitorUnixTop(&cycle, ch)
					}()
				}

				logrus.Debug("waiting for fetches to finish")
				innerWg.Wait()

				logrus.Infof("finished cycle %d, took %s", cycles, time.Since(start))
				cycle.Took = time.Since(start).String()
				if !cliCtx.Bool("not-in-sqlite") {
					err := db.CycleService.Update(cycle)
					if err != nil {
						logrus.WithError(err).Error("could not update cycle time to run.")
					}
				}
			}()
		}
	}
}

func deleteOldCyclesFromSqlite(retention time.Duration, db *data.DB) {
	timeToDelete := time.Now().Add(-retention)
	err := db.CycleService.DeleteAllBeforeTime(timeToDelete)
	if err != nil {
		logrus.WithError(err).Error("could not delete old cycles")
	}
}

func deleteOldCyclesFromDisk(retention time.Duration, outputDir string) {
	timeToDelete := time.Now().Add(-retention)
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "error occurred during output dir walking")
		}

		if filepath.Base(outputDir) == info.Name() {
			return nil
		}

		if info.IsDir() {
			empty, err := isDirEmpty(path)
			if err != nil {
				return errors.Wrapf(err, "could not determine if dir %q is empty", path)
			}

			if empty {
				if err := os.Remove(path); err != nil {
					return errors.Wrapf(err, "could not delete empty dir %q", path)
				}

				logrus.Debugf("deleted empty dir %q", path)

				return filepath.SkipDir
			}

			return nil
		}

		if info.ModTime().Before(timeToDelete) {
			if err := os.Remove(path); err != nil {
				return errors.Wrapf(err, "could not delete file %q", path)
			}
			logrus.Infof("deleted file %q", path)
			return nil
		}

		return nil
	})
	if err != nil {
		logrus.WithError(err).WithField("output_dir", outputDir).Error("could not walk in the output dir to delete old files")
	}
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, errors.Wrap(err, "could not determine if dir is empty")
}

func monitorMysqlProcessList(monitoringDb *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	logrus.Debug("fetching mysql process list")
	defer logrus.Debug("finished fetching mysql process list")

	processList, err := mysql.GetProcessList(monitoringDb)
	if err != nil {
		logrus.WithError(err).Error("could not get process list")
		return
	}

	ch <- data.CycleData{Cycle: cycle, MonitoredData: processList, Extension: "json", Path: "mysql/process-list"}
}

func monitorEngineINNODB(monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching engine innodb")
	defer logrus.Debug("finished fetching saving innodb")

	innodbStatusChan, err := mysql.GetEngineINNODBStatus(monitoringDB)
	if err != nil {
		logrus.WithError(err).Error("could not get innodb engine status")
		return
	}

	ch <- data.CycleData{Cycle: cycle, Extension: "txt", MonitoredData: innodbStatusChan, Path: "mysql/engine-innodb"}
}

func monitorSlaveStatus(monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching slave status")
	defer logrus.Debug("finished fetching slave status")

	slaveStatusChan, err := mysql.GetSlaveStatus(monitoringDB)
	if err != nil {
		logrus.WithError(err).Error("could not get slave status")
		return
	}

	ch <- data.CycleData{Cycle: cycle, Extension: "json", Path: "mysql/slave-status", MonitoredData: slaveStatusChan}
}

func monitorThreads(monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching threads")
	defer logrus.Debug("finished fetching threads")

	threadsChan, err := mysql.GetThreads(monitoringDB)
	if err != nil {
		logrus.WithError(err).Error("could not get slave status")
		return
	}

	ch <- data.CycleData{Cycle: cycle, MonitoredData: threadsChan, Extension: "json", Path: "mysql/ps_threads"}
}

func monitorUnixProcessList(cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching unix process list")
	defer logrus.Debug("finished fetching unix process list")

	buf, err := unix.GetProcessList()
	if err != nil {
		logrus.WithError(err).Error("could not get unix process list")
		return
	}

	innerCh := make(chan *data.MonitoredData)

	go func() {
		defer close(innerCh)
		innerCh <- &data.MonitoredData{UnixProcessList: &data.UnixProcessList{Text: buf.String()}}
	}()

	ch <- data.CycleData{Cycle: cycle, Extension: "txt", Path: "system/ps", MonitoredData: innerCh}
}

func monitorUnixTop(cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching unix top")
	defer logrus.Debug("finished fetching unix top")

	buf, err := unix.GetTop()
	if err != nil {
		logrus.WithError(err).Error("could not get unix top")
		return
	}

	innerCh := make(chan *data.MonitoredData)

	go func() {
		defer close(innerCh)
		innerCh <- &data.MonitoredData{Top: &data.Top{Text: buf.String()}}
	}()

	ch <- data.CycleData{Cycle: cycle, Extension: "txt", Path: "system/top", MonitoredData: innerCh}
}

func saveCycleToDB(db *data.DB, cycle *data.Cycle, ch chan *data.MonitoredData) {
	logrus.Debug("looping to save data in db")
	defer logrus.Debug("done looping to save data in db")
	for e := range ch {
		switch true {
		case e.MysqlProcessList != nil:
			e.MysqlProcessList.CycleID = cycle.ID
			err := db.MysqlProcessListService.Save(e.MysqlProcessList)
			if err != nil {
				logrus.WithError(err).Error("could not save mysql process list to db")
			}
		case e.EngineINNODBStatus != nil:
			e.EngineINNODBStatus.CycleID = cycle.ID
			err := db.EngineInnoDBService.Save(e.EngineINNODBStatus)
			if err != nil {
				logrus.WithError(err).Error("could not save mysql enigine inno db to db")
			}
		case e.SlaveStatus != nil:
			e.SlaveStatus.CycleID = cycle.ID
			err := db.SlaveStatusService.Save(e.SlaveStatus)
			if err != nil {
				logrus.WithError(err).Error("could not save mysql slave status to db")
			}
		case e.Thread != nil:
			e.Thread.CycleID = cycle.ID
			err := db.ThreadsService.Save(e.Thread)
			if err != nil {
				logrus.WithError(err).Error("could not save mysql thread to db")
			}
		case e.UnixProcessList != nil:
			e.UnixProcessList.CycleID = cycle.ID
			err := db.UnixProcessListService.Save(e.UnixProcessList)
			if err != nil {
				logrus.WithError(err).Error("could not save system processlist to db")
			}
		case e.Top != nil:
			e.Top.CycleID = cycle.ID
			err := db.TopService.Save(e.Top)
			if err != nil {
				logrus.WithError(err).Error("could not save system top to db")
			}
		default:
			logrus.Errorf("could not determine how to save %+v to db", e)
		}
	}
}

func saveCycleToFile(db *data.DB, outputDir, path, extension string, cycle *data.Cycle, d chan *data.MonitoredData) error {
	logrus.Debug("looping to save data in db")
	defer logrus.Debug("done looping to save data in db")
	abs, err := filepath.Abs(fmt.Sprintf("%s/%s", outputDir, path))
	if err != nil {
		return errors.Wrapf(err, "could not determine abs path to save file for %q", path)
	}

	filePath := fmt.Sprintf("%s/%s/%s00", abs, cycle.RanAt.Format("06-01-02"), cycle.RanAt.Format("15"))
	os.MkdirAll(filePath, os.ModePerm)

	buf := bytes.NewBufferString("")

	switch extension {
	case "json":
		enc := json.NewEncoder(buf)
		io.WriteString(buf, "[")

		encodeJson(enc, <-d)

		for e := range d {
			io.WriteString(buf, ",")
			encodeJson(enc, e)
		}

		io.WriteString(buf, "]")
	case "txt":
		encodeTxt(buf, <-d)
	default:
		return fmt.Errorf("could not determine how to save file with extension %s", extension)
	}

	absFilePath := fmt.Sprintf("%s/%s.%s.gz", filePath, cycle.RanAt.Format("2006-01-02T150405Z0700"), extension)

	logrus.Debugf("saving to %q", absFilePath)

	err = gzips.WriteToGzipper(buf, absFilePath)

	return errors.Wrap(err, "could not write buff to gzipper")
}

func encodeJson(enc *json.Encoder, d *data.MonitoredData) {
	if d == nil {
		return
	}

	switch true {
	case d.MysqlProcessList != nil:
		enc.Encode(d.MysqlProcessList)
	case d.Thread != nil:
		enc.Encode(d.Thread)
	case d.SlaveStatus != nil:
		enc.Encode(d.SlaveStatus)
	default:
		logrus.Errorf("could not determine how to decode json for %+v", d)
	}
}

func encodeTxt(buf *bytes.Buffer, d *data.MonitoredData) {
	if d == nil {
		return
	}

	switch true {
	case d.Top != nil:
		buf.WriteString(d.Top.Text)
	case d.UnixProcessList != nil:
		buf.WriteString(d.UnixProcessList.Text)
	case d.EngineINNODBStatus != nil:
		buf.WriteString(d.EngineINNODBStatus.Status)
	default:
		logrus.Errorf("could not determine how to decode json for %+v", d)
	}
}
