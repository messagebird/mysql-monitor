package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/messagebird/mysql-monitor/internal/api"
	"github.com/messagebird/mysql-monitor/internal/data"
	"github.com/messagebird/mysql-monitor/internal/gzips"
	"github.com/messagebird/mysql-monitor/internal/logging"
	"github.com/messagebird/mysql-monitor/internal/mysql"
	"github.com/messagebird/mysql-monitor/internal/unix"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const timeFormat = "2006-01-02T150405Z0700"

var (
	cntCycles = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mysql_monitor_cycels_total",
		Help: "The total number of cycles",
	})
	sumTimeToFetchMysql = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "mysql_monitor_time_to_fetch_from_mysql_seconds",
		Help: "The time it took to fetch the data from mysql in seconds",
	},
	[]string{"query_type"},
	)
	sumTimeToFetchSystem = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "mysql_monitor_time_to_fetch_from_system_seconds",
		Help: "The time it took to fetch the data from system in seconds",
	},
	[]string{"query_type"},
	)
	cntFetchDeadline = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mysql_monitor_ctx_fetch_deadline_exceeded_total",
		Help: "The total number of times teh deadline has been exceeded to fetch logging data",
	})
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
		cli.StringFlag{
			Name:  "metrics-port, mp",
			Usage: "On which port the metrics (/metrics) server should be exposed.",
			Value: "9100",
		},
	}
}

func Monitor(cliCtx *cli.Context) {
	logrus.Infof("hi, going to run this tool with an interval of %q", cliCtx.Duration("n"))

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	g, ctx := errgroup.WithContext(ctx)

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

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	g.Go(func() error {
		<-c
		logrus.Info("exiting....")
		cancelFunc()
		return nil
	})

	g.Go(func() error {
		ticker := time.Tick(time.Minute)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker:
				logrus.Debug("running log deletion")
				if cliCtx.Bool("not-in-sqlite") {
					deleteOldCyclesFromDisk(cliCtx.Duration("r"), cliCtx.String("o"))
				} else {
					deleteOldCyclesFromSqlite(cliCtx.Duration("r"), db)
				}
				logrus.Debug("finished log deletion")
			}
		}
	})

	if cliCtx.Bool("not-in-sqlite") {
		runMonitor(cliCtx, monitorDb, db, g, ctx)
	} else {
		runMonitorWithViewer(cliCtx, monitorDb, db, g, ctx)
	}

	err = g.Wait()
	if err != nil {
		logrus.WithError(err).Fatal("monitor crashed")
	}
}

func runMonitorWithViewer(cliCtx *cli.Context, monitorDb *sqlx.DB, db *data.DB, g *errgroup.Group, ctx context.Context) {
	runMonitor(cliCtx, monitorDb, db, g, ctx)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Mount("/api/v1/", api.NewRouter(db))

	g.Go(func() error {
		errCh := make(chan error)

		go func() {
			errCh <- errors.WithStack(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", cliCtx.String("p")), r))
		}()

		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-errCh:
				return errors.Wrap(err, "http server stopped")
			}
		}
	})
}

func runMonitor(cliCtx *cli.Context, monitorDb *sqlx.DB, db *data.DB, g *errgroup.Group, ctx context.Context) {
	cycles := 0

	innerCtx, cancel := context.WithCancel(context.Background())
	ch := make(chan data.CycleData)

	g.Go(func() error {
		logrus.Info("started metrics http server")
		r := chi.NewRouter()
		r.Handle("/metrics", promhttp.Handler())

		errCh := make(chan error)

		go func() {
			errCh <- errors.WithStack(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", cliCtx.String("mp")), r))
		}()

		for {
			select {
			case <-ctx.Done():
				return nil
			case err := <-errCh:
				return errors.Wrap(err, "http metrics server stopped")
			}
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-innerCtx.Done():
				return nil
			case e := <-ch:
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
		}
	})

	g.Go(func() error {
		time.Sleep(time.Until(time.Now().Round(cliCtx.Duration("n")).Truncate(time.Millisecond)))
		ticker := time.Tick(cliCtx.Duration("n"))

		timeout := time.Duration(float64(cliCtx.Duration("n")) * 0.75)

		logrus.Infof("will cancel/timeout requests for data if it takes longer then %s to fetch", timeout)

		for {
			select {
			case <-ctx.Done():
				cancel()
				return nil
			case start := <-ticker:
				cntCycles.Inc()
				cycles++

				mysqlDone := make(chan bool, 1)
				systemDone := make(chan bool, 1)
				allDone := make(chan bool, 2)

				id := uuid.NewV4()
				cycle := data.Cycle{ID: id, Cycle: cycles, RanAt: start.Round(cliCtx.Duration("n")).Truncate(time.Millisecond)}

				logrus.Infof("running cycle %d", cycles)

				if !cliCtx.Bool("not-in-sqlite") {
					logrus.Debug("saving cycle")
					err := db.CycleService.Save(cycle)
					if err != nil {
						logrus.WithError(err).Fatal("could not save cycle info")
						continue
					}
					logrus.Debug("done saving cycle")
				}

				ctx, _ := context.WithTimeout(ctx, timeout)

				go fetchMysql(ctx, cliCtx, monitorDb, &cycle, allDone, mysqlDone, ch)
				go fetchSystem(cliCtx, &cycle, allDone, systemDone, ch)

				waitForFetchToFinish(ctx, mysqlDone, systemDone, allDone, timeout)

				timeTook := time.Since(start)

				logrus.Infof("cycle %d, timestamp: %s", cycles, cycle.RanAt.Format(timeFormat))
				logrus.Infof("finished cycle %d, took %s", cycles, timeTook)
				cycle.Took = timeTook.String()
				if !cliCtx.Bool("not-in-sqlite") {
					err := db.CycleService.Update(cycle)
					if err != nil {
						logrus.WithError(err).Error("could not update cycle time to run.")
					}
				}
			}
		}
	})
}

func fetchMysql(ctx context.Context, cliCtx *cli.Context, monitorDb *sqlx.DB, cycle *data.Cycle, allDone chan bool, mysqlDone chan bool, ch chan data.CycleData) {
		defer func() { allDone <- true }()
		defer func() { mysqlDone <- true }()
		var innerWg sync.WaitGroup

		if !cliCtx.Bool("exclude-mysql-process-list") {
			innerWg.Add(1)
			go func() {
				defer innerWg.Done()
				timer := prometheus.NewTimer(sumTimeToFetchMysql.WithLabelValues("mysql-process-list"))
				defer timer.ObserveDuration()

				monitorMysqlProcessList(ctx, monitorDb, cycle, ch)
			}()
		}

		if !cliCtx.Bool("exclude-innodb-status") {
			innerWg.Add(1)
			go func() {
				defer innerWg.Done()
				timer := prometheus.NewTimer(sumTimeToFetchMysql.WithLabelValues("innodb-status"))
				defer timer.ObserveDuration()

				monitorEngineINNODB(ctx, monitorDb, cycle, ch)
			}()
		}

		if !cliCtx.Bool("exclude-slave-status") {
			innerWg.Add(1)
			go func() {
				defer innerWg.Done()
				timer := prometheus.NewTimer(sumTimeToFetchMysql.WithLabelValues("slave-status"))
				defer timer.ObserveDuration()

				monitorSlaveStatus(ctx, monitorDb, cycle, ch)
			}()
		}

		if !cliCtx.Bool("exclude-ps-threads") {
			innerWg.Add(1)
			go func() {
				defer innerWg.Done()
				timer := prometheus.NewTimer(sumTimeToFetchMysql.WithLabelValues("ps-threads"))
				defer timer.ObserveDuration()

				monitorThreads(ctx, monitorDb, cycle, ch)
			}()
		}

		logrus.Debug("waiting for mysql fetches to finish")
		innerWg.Wait()
}

func fetchSystem(cliCtx *cli.Context, cycle *data.Cycle, allDone chan bool, systemDone chan bool, ch chan data.CycleData) {
	defer func() { allDone <- true }()
	defer func() { systemDone <- true }()
	var innerWg sync.WaitGroup

	if !cliCtx.Bool("exclude-unix-process-list") {
		innerWg.Add(1)
		go func() {
			defer innerWg.Done()
			timer := prometheus.NewTimer(sumTimeToFetchSystem.WithLabelValues("process-list"))
			defer timer.ObserveDuration()

			monitorUnixProcessList(cycle, ch)
		}()
	}

	if !cliCtx.Bool("exclude-unix-top") {
		innerWg.Add(1)
		go func() {
			defer innerWg.Done()
			timer := prometheus.NewTimer(sumTimeToFetchMysql.WithLabelValues("top"))
			defer timer.ObserveDuration()

			monitorUnixTop(cycle, ch)
		}()
	}

	logrus.Debug("waiting for system fetches to finish")
	innerWg.Wait()
}

func waitForFetchToFinish(ctx context.Context, mysqlDone chan bool, systemDone chan bool, allDone chan bool, timeout time.Duration) {
	allDoneCount := 0
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				continue
			}
			logrus.WithError(ctx.Err()).Warnf("partial data stored, one of the systems that is being logged did not finish. It took longer then %s to fetch the data.", timeout)
			cntFetchDeadline.Inc()
			return
		case <-mysqlDone:
			logrus.Info("mysql logs fetched")
		case <-systemDone:
			logrus.Info("system logs fetched")
		case <-allDone:
			allDoneCount++

			if allDoneCount == 2 {
				logrus.Info("all logs fetched")
				return
			}
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

func monitorMysqlProcessList(ctx context.Context, monitoringDb *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logging.Trace(logging.TraceTypeEntering)
	defer logging.Trace(logging.TraceTypeExiting)

	logrus.Debug("fetching mysql process list")
	defer logrus.Debug("finished fetching mysql process list")

	processList, err := mysql.GetProcessList(ctx, monitoringDb)
	if err != nil {
		logrus.WithError(err).Error("could not get process list")
		return
	}

	ch <- data.CycleData{Cycle: cycle, MonitoredData: processList, Extension: "json", Path: "mysql/process-list"}
}

func monitorEngineINNODB(ctx context.Context, monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching engine innodb")
	defer logrus.Debug("finished fetching saving innodb")

	innodbStatusChan, err := mysql.GetEngineINNODBStatus(ctx, monitoringDB)
	if err != nil {
		logrus.WithError(err).Error("could not get innodb engine status")
		return
	}

	ch <- data.CycleData{Cycle: cycle, Extension: "txt", MonitoredData: innodbStatusChan, Path: "mysql/engine-innodb"}
}

func monitorSlaveStatus(ctx context.Context, monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching slave status")
	defer logrus.Debug("finished fetching slave status")

	slaveStatusChan, err := mysql.GetSlaveStatus(ctx, monitoringDB)
	if err != nil {
		logrus.WithError(err).Error("could not get slave status")
		return
	}

	ch <- data.CycleData{Cycle: cycle, Extension: "json", Path: "mysql/slave-status", MonitoredData: slaveStatusChan}
}

func monitorThreads(ctx context.Context, monitoringDB *sqlx.DB, cycle *data.Cycle, ch chan data.CycleData) {
	logrus.Debug("fetching threads")
	defer logrus.Debug("finished fetching threads")

	threadsChan, err := mysql.GetThreads(ctx, monitoringDB)
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

	absFilePath := fmt.Sprintf("%s/%s.%s.gz", filePath, cycle.RanAt.Format(timeFormat), extension)

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
