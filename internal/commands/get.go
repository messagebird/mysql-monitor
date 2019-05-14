package commands

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/messagebird/mysql-monitor/internal/data"
	"log"
	"net/http"
	"os"
	"strconv"
)

func BuildGetCommand() cli.Command {
	return cli.Command{
		Name:    "get",
		Aliases: []string{"g"},
		Usage:   "view the gathered data",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "api-url, u",
				Usage: "the url to where the mysql monitor api is hosted",
				Value: "http://localhost:9090",
			},
		},
		Subcommands: cli.Commands{
			cli.Command{
				Name:        "cycles",
				Aliases:     []string{"c"},
				Description: "get the available cycles",
				Action:      getCycles,
			},
			cli.Command{
				Name:    "mysql",
				Aliases: []string{"m"},
				Subcommands: cli.Commands{
					cli.Command{
						Name:    "process-list",
						Usage:   "ps {cycle_id}",
						Aliases: []string{"pl"},
						Action:  getMysqlProcessList,
					},
					cli.Command{
						Name:    "engine-inno-db",
						Usage:   "eid {cycle_id}",
						Aliases: []string{"eid"},
						Action:  getMysqlEngineInnoDb,
					},
					cli.Command{
						Name:    "slave-status",
						Usage:   "ss {cycle_id}",
						Aliases: []string{"ss"},
						Action:  getMysqlSlaveStatus,
					},
					cli.Command{
						Name:    "threads",
						Usage:   "tr {cycle_id}",
						Aliases: []string{"tr"},
						Action:  getMysqlThreads,
					},
				},
			},
			cli.Command{
				Name: "system",
				Aliases: []string{"s"},
				Subcommands: cli.Commands{
					cli.Command{
						Name: "top",
						Aliases: []string{"t"},
						Action: getSystemTop,
					},
					cli.Command{
						Name: "process-list",
						Aliases: []string{"pl"},
						Action: getSystemProcessList,
					},
				},
			},
		},
	}
}

func getCycles(ctx *cli.Context) {
	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles", ctx.Parent().String("u")))
	if err != nil {
		logrus.WithError(err).Error("could not get cycles")
		return
	}

	var b []data.Cycle
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Id", "Ran at", "Took"})
	table.SetColWidth(40)

	for _, e := range b {
		table.Append([]string{e.ID.String(), e.RanAt.String(), e.Took})
	}

	table.Render()
}

func getMysqlProcessList(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/mysql/process-list", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not mysql process list")
		return
	}

	var b []data.MysqlProcessList
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Id", "user", "host", "db", "command", "time", "state", "info", "rows sent", "rows examined"})
	table.SetRowLine(true)
	for _, e := range b {
		table.Append([]string{strconv.Itoa(int(e.ID)), e.User, e.Host, e.DB.String, e.Command, strconv.Itoa(int(e.Time)), e.State.String, e.Info.String, strconv.Itoa(int(e.RowsSent)), strconv.Itoa(int(e.RowsExamined))})
	}

	table.Render()
}

func getMysqlEngineInnoDb(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/mysql/engine-innodb-status", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not mysql engine inno db")
		return
	}

	var b []data.EngineINNODBStatus
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	for _, e := range b {
		log.SetFlags(0)
		log.Println(e.Status)
	}
}

func getMysqlSlaveStatus(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/mysql/slave-status", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not mysql engine inno db")
		return
	}

	var b []data.SlaveStatus
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	err = json.NewEncoder(os.Stdout).Encode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}
}

func getMysqlThreads(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/mysql/threads", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not mysql threads")
		return
	}

	var b []data.Thread
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	err = json.NewEncoder(os.Stdout).Encode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}
}

func getSystemTop(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/system/top", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not get system top")
		return
	}

	var b []data.Top
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	for _, e := range b {
		log.SetFlags(0)
		log.Println(e.Text)
	}

	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}
}

func getSystemProcessList(ctx *cli.Context) {
	if ctx.Args().Get(0) == "" || uuid.FromStringOrNil(ctx.Args().Get(0)) == uuid.Nil {
		logrus.Fatalf("%q is not a valid cycle id", ctx.Args().Get(0))
	}

	res, err := http.Get(fmt.Sprintf("%s/api/v1/cycles/%s/system/process-list", ctx.Parent().Parent().String("u"), ctx.Args().Get(0)))
	if err != nil {
		logrus.WithError(err).Error("could not get system process list")
		return
	}

	var b []data.UnixProcessList
	err = json.NewDecoder(res.Body).Decode(&b)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}

	for _, e := range b {
		log.SetFlags(0)
		log.Println(e.Text)
	}

	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}
}
