package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/messagebird/mysql-monitor/internal/commands"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func setupLogging(lvl string) {
	logLvl, err := logrus.ParseLevel(lvl)
	if err != nil {
		logrus.WithError(err).Warning("could not parse log level")
	} else {
		logrus.SetLevel(logLvl)
	}

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableLevelTruncation: true, QuoteEmptyFields: true})
}

var Version string
var Commit string

func main() {
	app := cli.NewApp()
	app.Name = "mysql monitor"
	app.Usage = "monitors mysql systems"
	app.Description = "monitors mysql systems"
	app.Version = fmt.Sprintf("Version: %s\n\t Commit: %s", Version, Commit)
	app.Before = func(ctx *cli.Context) error {
		setupLogging(ctx.String("level"))

		if ctx.Bool("trace") {
			go http.ListenAndServe(":6060", http.DefaultServeMux)
		}

		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "level",
			Usage: "log level",
			Value: "info",
		},
		cli.BoolFlag{
			Name: "trace",
			Usage: "if golang pprof should be enabled",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "monitor",
			Aliases: []string{"m"},
			Usage:   "runs the monitor",
			Flags:   commands.GetMonitorCommandFlags(),
			Action:  commands.Monitor,
		},
		commands.BuildGetCommand(),
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.WithError(err).Fatal(err.Error())
	}
	logrus.Info("good bye")
}
