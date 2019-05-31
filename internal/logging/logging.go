//go:generate go-enum -f=$GOFILE

package logging

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"runtime"
)

// Trace prints the calling function file, name and line.
func Trace(traceType TraceType) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	logrus.Tracef("%s %s,:%d %s\n", traceType.String(), frame.File, frame.Line, frame.Function)
}

// TraceType explains if the trace is entering a function, is exciting a function or if its still in a function.
/*
ENUM(
entering
inside
exiting
)
*/
type TraceType int

var (
	cntError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mysql_monitor_error_count_total",
		Help: "The amount of generic errors that happened",
	})
)

type PrometheusHook struct {
}

func (*PrometheusHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

func (*PrometheusHook) Fire(*logrus.Entry) error {
	cntError.Inc()
	return nil
}
