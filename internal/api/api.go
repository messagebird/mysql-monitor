package api

import (
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/http-swagger"
	_ "github.com/messagebird/mysql-monitor/docs"
	"github.com/messagebird/mysql-monitor/internal/data"
	"io"
	"net/http"
	"time"
)

// @title Mysql monitor
// @version 1.0
// @description This api can be used to view the logs collected by the mysql monitor

// @host localhost:9090
// @BasePath /api/v1
func NewRouter(db *data.DB) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/cycles", func(r chi.Router) {
		r.Use(middleware.SetHeader("Content-Type", "application/json"))
		r.Use(middleware.Timeout(time.Second))

		r.Get("/", getCycles(db))

		r.Route("/{cycle_id}", func(r chi.Router) {
			r.Route("/mysql", func(r chi.Router) {
				r.Get("/process-list", getMysqlProcessList(db))
				r.Get("/slave-status", getMysqlSlaveStatus(db))
				r.Get("/engine-innodb-status", getMysqlEngineInnoDBStatus(db))
				r.Get("/threads", getMysqlThreads(db))
			})
			r.Route("/system", func(r chi.Router) {
				r.Get("/top", getSystemTop(db))
				r.Get("/process-list", getSystemProcessList(db))
			})
		})
	})

	return r
}

// GetCycles Returns a list containing all the available cycles.
// @Summary Get all cycles
// @Description Get all cycles that are available
// @ID get-all-cycles
// @Tags mysql
// @Accept  json
// @Produce  json
// @Success 200 {object} data.Cycle
// @Router /cycles [get]
func getCycles(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.CycleService.GetAll()
		if err != nil {
			logrus.WithError(err).Error("could not get all cycles for api")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetMysqlProcessList Returns a list of mysql processlist belonging to a cycle id.
// @Summary Get all mysql processlist by cycle id
// @Description Get a list of mysql processlist belonging to a cycle id.
// @ID get-all-mysql-process-list
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Tags mysql
// @Success 200 {object} data.MysqlProcessList
// @Router /cycles/{cycle_id}/mysql/processlist [get]
func getMysqlProcessList(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.MysqlProcessListService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all mysql process list for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetMysqlProcessList Returns a list of mysql slave status.
// @Summary Get all mysql slave statuses by cycle id
// @Description Get a list of mysql slave status belonging to a cycle id.
// @ID get-all-mysql-slave-status
// @Tags mysql
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} data.SlaveStatus
// @Router /cycles/{cycle_id}/mysql/slave-status [get]
func getMysqlSlaveStatus(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.SlaveStatusService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all mysql slave status for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetMysqlEngineInnoDBStatus Returns a list of mysql engine innodb statuses.
// @Summary Get all mysql engine InnoDB status by cycle id
// @Description Get a list of mysql engine innoDB statuses belonging to a cycle id.
// @ID get-all-mysql-engine-innodb
// @Tags mysql
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} data.SlaveStatus
// @Router /cycles/{cycle_id}/mysql/engine-innodb-status [get]
func getMysqlEngineInnoDBStatus(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.EngineInnoDBService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all engine innodb for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetMysqlThreads Returns a list of mysql threads.
// @Summary Get all mysql threads by cycle id
// @Description Get a list of mysql threads belonging to a cycle id.
// @ID get-all-mysql-threads
// @Tags mysql
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} data.Thread
// @Router /cycles/{cycle_id}/mysql/threads [get]
func getMysqlThreads(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.ThreadsService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all mysql threads for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetSystemTop Returns a list of system top this is always 1 entry.
// @Summary Get all system top by cycle id, this is always 1 entry
// @Description Get a list of system top belonging to a cycle id, this is always 1 entry.
// @ID get-all-system-top
// @Tags system
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} data.Top
// @Router /cycles/{cycle_id}/system/top [get]
func getSystemTop(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.TopService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all mysql threads for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}

// GetSystemProcessList Returns a list of system processlist this is always 1 entry.
// @Summary Get all system processlist by cycle id, this is always 1 entry
// @Description Get a list of system processlist belonging to a cycle id, this is always 1 entry.
// @ID get-all-system-process-list
// @Tags system
// @Param cycle_id path string true "Cycle ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} data.UnixProcessList
// @Router /cycles/{cycle_id}/system/process-list [get]
func getSystemProcessList(db *data.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ch, err := db.UnixProcessListService.GetAllByCycle(uuid.FromStringOrNil(chi.URLParam(r, "cycle_id")))
		if err != nil {
			logrus.WithError(err).Errorf("could not get all mysql threads for cycle %s for api", chi.URLParam(r, "cycle_id"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		_, _ = io.WriteString(w, "[")
		_ = enc.Encode(<-ch)

		for e := range ch {
			_, _ = io.WriteString(w, ",")
			_ = enc.Encode(e)
		}

		_, _ = io.WriteString(w, "]")
	}
}
