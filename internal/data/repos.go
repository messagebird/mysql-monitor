package data

import (
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"time"
)

type CycleService struct {
	db *DB
}

func (c CycleService) GetAll() (chan Cycle, error) {
	c.db.lock.RLock()
	defer c.db.lock.RUnlock()

	rows, err := c.db.Queryx(`select id, ran_at, took, cycle from cycle`)
	if err != nil {
		return nil, errors.Wrap(err, "could not save cycle into DB.")
	}

	ch := make(chan Cycle)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c Cycle
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan cycle db row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

func (c CycleService) Save(cycle Cycle) error {
	c.db.lock.Lock()
	logrus.Debug("cycle save got lock")
	defer logrus.Debug("cycle save released lock")
	defer c.db.lock.Unlock()

	_, err := c.db.NamedExec(`insert into cycle (id, ran_at, took, cycle) VALUES (:id, :ran_at, :took, :cycle)`, cycle)
	if err != nil {
		return errors.Wrap(err, "could not save cycle into DB.")
	}

	return nil
}

func (c CycleService) Update(cycle Cycle) error {
	c.db.lock.Lock()
	defer c.db.lock.Unlock()

	_, err := c.db.NamedExec(`update cycle set took = :took where id = :id`, cycle)
	if err != nil {
		return errors.Wrap(err, "could not update cycle into DB.")
	}

	return nil
}

func (c CycleService) DeleteAllBeforeTime(timeToDelete time.Time) error {
	c.db.lock.Lock()
	defer c.db.lock.Unlock()

	res, err := c.db.Exec(`delete from cycle where ran_at < ?`, timeToDelete)
	if err != nil {
		return errors.Wrap(err, "could not deleted old cycles from DB.")
	}

	r, _ := res.RowsAffected()
	if r > 0 {
		logrus.Infof("deleted %d cycles older then %q", r, timeToDelete)
	}

	return nil
}
func (c CycleService) GetAllBeforeTime(timeToDelete time.Time) (chan Cycle, error) {
	logrus.Debug("cycle service get all before got lock")
	c.db.lock.RLock()
	defer logrus.Debug("cycle service get all before released lock")
	defer c.db.lock.RUnlock()

	rows, err := c.db.Query(`select id from cycle where ran_at < ?`, timeToDelete)
	if err != nil {
		return nil, errors.Wrap(err, "could not deleted old cycles from DB.")
	}

	ch := make(chan Cycle)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c Cycle
			err := rows.Scan(&c.ID)
			if err != nil {
				logrus.WithError(err).Error("could not scan cycle to struct")
				continue
			}
			ch <- c
		}
	}()

	return ch ,nil
}

type SlaveStatusService struct {
	db *DB
}

func (s SlaveStatusService) GetAllByCycle(cycleID uuid.UUID) (chan SlaveStatus, error) {
	s.db.lock.RLock()
	defer s.db.lock.RUnlock()

	rows, err := s.db.Queryx(`select id,
       cycle_id,
       Master_Log_File,
       Master_Host,
       Master_User,
       Master_Port,
       Connect_Retry,
       Read_Master_Log_Pos,
       Relay_Log_File,
       Relay_Log_Pos,
       Relay_Master_Log_File,
       Slave_IO_Running,
       Slave_IO_State,
       Slave_IO_Status,
       Slave_SQL_Running,
       Replicate_Do_DB,
       Replicate_Ignore_DB,
       Replicate_Do_Table,
       Replicate_Ignore_Table,
       Replicate_Wild_Do_Table,
       Replicate_Wild_Ignore_Table,
       Last_Errno,
       Last_Error,
       Skip_Counter,
       Exec_Master_Log_Pos,
       Relay_Log_Space,
       Until_Condition,
       Until_Log_File,
       Until_Log_Pos,
       Master_SSL_Allowed,
       Master_SSL_CA_File,
       Master_SSL_CA_Path,
       Master_SSL_Cert,
       Master_SSL_Cipher,
       Master_SSL_Key,
       Seconds_Behind_Master,
       Master_SSL_Verify_Server_Cert,
       Last_IO_Errno,
       Last_IO_Error,
       Last_SQL_Errno,
       Last_SQL_Error,
       Replicate_Ignore_Server_Ids,
       Master_Server_Id,
       Master_UUID,
       Master_Info_File,
       SQL_Delay,
       SQL_Remaining_Delay,
       Slave_SQL_Running_State,
       Slave_Status,
       Master_Retry_Count,
       Master_Bind,
       Last_IO_Error_Timestamp,
       Last_SQL_Error_Timestamp,
       Master_SSL_Crl,
       Master_SSL_Crlpath,
       Retrieved_Gtid_Set,
       Executed_Gtid_Set,
       Auto_Position,
       Replicate_Rewrite_DB,
       Channel_Name,
       Master_TLS_Version
from slave_status
where cycle_id = ?;
`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get all mysql slave status by cycle")
	}

	ch := make(chan SlaveStatus)

	go func() {
		defer close(ch)
		for rows.Next() {
			var status SlaveStatus
			err = rows.StructScan(&status)
			if err != nil {
				logrus.WithError(err).Error("could not scan mysql slave status db row to struct")
			}

			ch <- status
		}
	}()

	return ch, nil
}

func (s SlaveStatusService) Save(status *SlaveStatus) error {
	s.db.lock.Lock()
	defer s.db.lock.Unlock()

	_, err := s.db.NamedExec(`
insert into slave_status (Master_Log_File, Master_Host, Master_User, Master_Port, Connect_Retry, Read_Master_Log_Pos,
                          Relay_Log_File, Relay_Log_Pos, Relay_Master_Log_File, Slave_IO_Running, Slave_IO_State,
                          Slave_IO_Status, Slave_SQL_Running, Replicate_Do_DB, Replicate_Ignore_DB, Replicate_Do_Table,
                          Replicate_Ignore_Table, Replicate_Wild_Do_Table, Replicate_Wild_Ignore_Table, Last_Errno,
                          Last_Error, Skip_Counter, Exec_Master_Log_Pos, Relay_Log_Space, Until_Condition,
                          Until_Log_File, Until_Log_Pos, Master_SSL_Allowed, Master_SSL_CA_File, Master_SSL_CA_Path,
                          Master_SSL_Cert, Master_SSL_Cipher, Master_SSL_Key, Seconds_Behind_Master,
                          Master_SSL_Verify_Server_Cert, Last_IO_Errno, Last_IO_Error, Last_SQL_Errno, Last_SQL_Error,
                          Replicate_Ignore_Server_Ids, Master_Server_Id, Master_UUID, Master_Info_File, SQL_Delay,
                          SQL_Remaining_Delay, Slave_SQL_Running_State, Slave_Status, Master_Retry_Count, Master_Bind,
                          Last_IO_Error_Timestamp, Last_SQL_Error_Timestamp, Master_SSL_Crl, Master_SSL_Crlpath,
                          Retrieved_Gtid_Set, Executed_Gtid_Set, Auto_Position, Replicate_Rewrite_DB, Channel_Name,
                          Master_TLS_Version, cycle_id)
values (:Master_Log_File, :Master_Host, :Master_User, :Master_Port, :Connect_Retry, :Read_Master_Log_Pos,
                          :Relay_Log_File, :Relay_Log_Pos, :Relay_Master_Log_File, :Slave_IO_Running, :Slave_IO_State,
                          :Slave_IO_Status, :Slave_SQL_Running, :Replicate_Do_DB, :Replicate_Ignore_DB, :Replicate_Do_Table,
                          :Replicate_Ignore_Table, :Replicate_Wild_Do_Table, :Replicate_Wild_Ignore_Table, :Last_Errno,
                          :Last_Error, :Skip_Counter, :Exec_Master_Log_Pos, :Relay_Log_Space, :Until_Condition,
                          :Until_Log_File, :Until_Log_Pos, :Master_SSL_Allowed, :Master_SSL_CA_File, :Master_SSL_CA_Path,
                          :Master_SSL_Cert, :Master_SSL_Cipher, :Master_SSL_Key, :Seconds_Behind_Master,
                          :Master_SSL_Verify_Server_Cert, :Last_IO_Errno, :Last_IO_Error, :Last_SQL_Errno, :Last_SQL_Error,
                          :Replicate_Ignore_Server_Ids, :Master_Server_Id, :Master_UUID, :Master_Info_File, :SQL_Delay,
                          :SQL_Remaining_Delay, :Slave_SQL_Running_State, :Slave_Status, :Master_Retry_Count, :Master_Bind,
                          :Last_IO_Error_Timestamp, :Last_SQL_Error_Timestamp, :Master_SSL_Crl, :Master_SSL_Crlpath,
                          :Retrieved_Gtid_Set, :Executed_Gtid_Set, :Auto_Position, :Replicate_Rewrite_DB, :Channel_Name,
                          :Master_TLS_Version, :cycle_id)
	`, status)
	if err != nil {
		return errors.Wrap(err, "could not save slave status into DB.")
	}

	return nil
}

type MysqlProcessListService struct {
	db *DB
}

func (m MysqlProcessListService) Save(list *MysqlProcessList) error {
	m.db.lock.Lock()
	defer m.db.lock.Unlock()

	_, err := m.db.NamedExec(`
insert into mysql_process_list (id, user, host, db, command, time, state, info, rows_sent, rows_examined, cycle_id)
VALUES (:id, :user, :host, :db, :command, :time, :state, :info, :rows_sent, :rows_examined, :cycle_id)
	`, list)
	if err != nil {
		return errors.Wrap(err, "could not save mysql process ist into DB.")
	}

	return nil
}

func (m MysqlProcessListService) GetAllByCycle(cycleID uuid.UUID) (chan MysqlProcessList, error) {
	m.db.lock.RLock()
	defer m.db.lock.RUnlock()

	rows, err := m.db.Queryx(`select itemId, cycle_id, id, user, host, db, command, time, state, info, rows_sent, rows_examined from mysql_process_list where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get all mysql processlist")
	}

	ch := make(chan MysqlProcessList)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c MysqlProcessList
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan cycle db row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

type EngineInnoDBService struct {
	db *DB
}

func (e EngineInnoDBService) GetAllByCycle(cycleID uuid.UUID) (chan EngineINNODBStatus, error) {
	e.db.lock.RLock()
	defer e.db.lock.RUnlock()

	rows, err := e.db.Queryx(`select id, cycle_id, name, type, status from engine_inno_db_status where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get all engine innodb by cycle id")
	}

	ch := make(chan EngineINNODBStatus)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c EngineINNODBStatus
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan engine innodb row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

func (e EngineInnoDBService) Save(status *EngineINNODBStatus) error {
	e.db.lock.Lock()
	defer e.db.lock.Unlock()

	_, err := e.db.NamedExec(`
insert into engine_inno_db_status (cycle_id, name, type, status) VALUES (:cycle_id, :name, :type, :status)
`, status)
	if err != nil {
		return errors.Wrap(err, "could not save engine inno db ist into DB.")
	}

	return nil
}

type ThreadsService struct {
	db *DB
}

func (t ThreadsService) GetAllByCycle(cycleID uuid.UUID) (chan Thread, error) {
	t.db.lock.RLock()
	defer t.db.lock.RUnlock()

	rows, err := t.db.Queryx(`select id, cycle_id, thread_id, name, type, processlist_id, processlist_user, processlist_host, processlist_db, processlist_command, processlist_time, processlist_state, processlist_info, parent_thread_id, role, instrumented, history, connection_type, thread_os_id from threads where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get all mysql threads by cycle")
	}

	ch := make(chan Thread)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c Thread
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan mysql thread row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

func (t ThreadsService) Save(thread *Thread) error {
	t.db.lock.Lock()
	defer t.db.lock.Unlock()

	_, err := t.db.NamedExec(`
insert into threads (cycle_id, thread_id, name, type, processlist_id, processlist_user, processlist_host,
                     processlist_db, processlist_command, processlist_time, processlist_state, processlist_info,
                     parent_thread_id, role, instrumented, history, connection_type, thread_os_id)
VALUES (:cycle_id, :thread_id, :name, :type, :processlist_id, :processlist_user, :processlist_host,
        :processlist_db, :processlist_command, :processlist_time, :processlist_state, :processlist_info,
        :parent_thread_id, :role, :instrumented, :history, :connection_type, :thread_os_id);
`, thread)
	if err != nil {
		return errors.Wrap(err, "could not save threads into DB.")
	}

	return nil
}

type UnixProcessListService struct {
	db *DB
}

func (u UnixProcessListService) GetAllByCycle(cycleID uuid.UUID) (chan UnixProcessList, error) {
	u.db.lock.RLock()
	defer u.db.lock.RUnlock()

	rows, err := u.db.Queryx(`select id, cycle_id, text from unix_process_list where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get all system processlist by cycle")
	}

	ch := make(chan UnixProcessList)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c UnixProcessList
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan system process list db row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

func (u UnixProcessListService) Save(list *UnixProcessList) error {
	u.db.lock.Lock()
	defer u.db.lock.Unlock()

	_, err := u.db.NamedExec(`
insert into unix_process_list (cycle_id, text) values (:cycle_id, :text);
`, list)
	if err != nil {
		return errors.Wrap(err, "could not save unix processlist into DB.")
	}

	return nil
}

type TopService struct {
	db *DB
}

func (t TopService) GetAllByCycle(cycleID uuid.UUID) (chan Top, error) {
	t.db.lock.RLock()
	defer t.db.lock.RUnlock()

	rows, err := t.db.Queryx(`select id, cycle_id, text from top where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not execute query to get system top by cycle id")
	}

	ch := make(chan Top)

	go func() {
		defer close(ch)
		for rows.Next() {
			var c Top
			err = rows.StructScan(&c)
			if err != nil {
				logrus.WithError(err).Error("could not scan system top row to struct")
			}

			ch <- c
		}
	}()

	return ch, nil
}

func (t TopService) Save(top *Top) error {
	t.db.lock.Lock()
	defer t.db.lock.Unlock()

	_, err := t.db.NamedExec(`
insert into top (cycle_id, text) values (:cycle_id, :text);
`, top)
	if err != nil {
		return errors.Wrap(err, "could not save unix processlist into DB.")
	}

	return nil
}

type FileOnDiskService struct {
	db *DB
}

func (f FileOnDiskService) Save(cycleID uuid.UUID, path string) error {
	logrus.Debug("saving file on disk")
	defer logrus.Debug("done saving file on disk")

	f.db.lock.Lock()
	logrus.Debug("file on disk service got db lock")
	defer logrus.Debug("file on disk service released db lock")
	defer f.db.lock.Unlock()

	_, err := f.db.Exec(`insert into file_on_disk (cycle_id, path) values (?, ?)`, cycleID, path)
	if err != nil {
		return errors.Wrap(err, "could not save file on disk in db.")
	}

	return nil
}

func (f FileOnDiskService) GetAllPathByCycleID(cycleID uuid.UUID) (chan string, error) {
	f.db.lock.RLock()
	logrus.Debug("get path by cycle got db lock")

	defer logrus.Debug("get path by cycle released db lock")
	defer f.db.lock.RUnlock()

	rows, err := f.db.Query(`select path from file_on_disk where cycle_id = ?`, cycleID)
	if err != nil {
		return nil, errors.Wrap(err, "could not get all paths by cycle id")
	}

	ch := make(chan string)

	go func() {
		defer close(ch)
		for rows.Next() {
			var s string
			err := rows.Scan(&s)
			if err != nil {
				logrus.WithError(err).Error("could not scan path to string")
				continue
			}

			ch <- s
		}
	}()

	return ch, nil
}
