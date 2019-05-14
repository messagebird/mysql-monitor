package data

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"time"
)

type CycleData struct {
	Cycle         *Cycle
	MonitoredData chan *MonitoredData
	Extension     string
	Path          string
}

type MonitoredData struct {
	SlaveStatus        *SlaveStatus
	MysqlProcessList   *MysqlProcessList
	EngineINNODBStatus *EngineINNODBStatus
	Thread             *Thread
	UnixProcessList    *UnixProcessList
	Top                *Top
}

type Cycle struct {
	Cycle int       `db:"cycle"json:"cycle"`
	ID    uuid.UUID `db:"id" json:"id"`
	RanAt time.Time `db:"ran_at" json:"ran_at"`
	Took  string    `db:"took" json:"took"`
}

func (c Cycle) String() string {
	return c.RanAt.String()
}

type SlaveStatus struct {
	ID      int       `json:"-"db:"id"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	MasterLogFile             string         `json:"master_log_file" db:"Master_Log_File"`
	MasterHost                string         `json:"master_host" db:"Master_Host"`
	MasterUser                string         `json:"master_user" db:"Master_User"`
	MasterPort                string         `json:"master_port" db:"Master_Port"`
	ConnectRetry              string         `json:"connect_retry" db:"Connect_Retry"`
	ReadMasterLogPos          int            `json:"read_master_log_pos" db:"Read_Master_Log_Pos"`
	RelayLogFile              string         `json:"relay_log_file" db:"Relay_Log_File"`
	RelayLogPos               int            `json:"relay_log_pos" db:"Relay_Log_Pos"`
	RelayMasterLogFile        string         `json:"relay_master_log_file" db:"Relay_Master_Log_File"`
	SlaveIoRunning            string         `json:"slave_io_running" db:"Slave_IO_Running"`
	SlaveIoState              string         `json:"slave_io_state" db:"Slave_IO_State"`
	SlaveIoStatus             string         `json:"slave_io_status" db:"Slave_IO_Status"`
	SlaveSqlRunning           string         `json:"slave_sql_running" db:"Slave_SQL_Running"`
	ReplicateDodb             sql.NullString `json:"replicate_do_db" db:"Replicate_Do_DB"swaggertype:"primitive,string"`
	ReplicateIgnoredb         sql.NullString `json:"replicate_ignore_db" db:"Replicate_Ignore_DB"swaggertype:"primitive,string"`
	ReplicateDoTable          sql.NullString `json:"replicate_do_table" db:"Replicate_Do_Table"swaggertype:"primitive,string"`
	ReplicateIgnoreTable      sql.NullString `json:"replicate_ignore_table" db:"Replicate_Ignore_Table"swaggertype:"primitive,string"`
	ReplicateWildDoTable      sql.NullString `json:"replicate_wild_do_table" db:"Replicate_Wild_Do_Table"swaggertype:"primitive,string"`
	ReplicateWildIgnoreTable  sql.NullString `json:"replicate_wild_ignore_table" db:"Replicate_Wild_Ignore_Table"swaggertype:"primitive,string"`
	LastErrno                 int            `json:"last_errno" db:"Last_Errno"`
	LastError                 sql.NullString `json:"last_error" db:"Last_Error"swaggertype:"primitive,string"`
	SkipCounter               int            `json:"skip_counter" db:"Skip_Counter"`
	ExecMasterLogPos          int            `json:"exec_master_log_pos" db:"Exec_Master_Log_Pos"`
	RelayLogSpace             int            `json:"relay_log_space" db:"Relay_Log_Space"`
	UntilCondition            string         `json:"until_condition" db:"Until_Condition"`
	UntilLogFile              string         `json:"until_log_file" db:"Until_Log_File"`
	UntilLogPos               int            `json:"until_log_pos" db:"Until_Log_Pos"`
	MasterSslAllowed          string         `json:"master_ssl_allowed" db:"Master_SSL_Allowed"`
	MasterSslCaFile           sql.NullString `json:"master_ssl_ca_file" db:"Master_SSL_CA_File"swaggertype:"primitive,string"`
	MasterSslCaPath           sql.NullString `json:"master_ssl_ca_path" db:"Master_SSL_CA_Path"swaggertype:"primitive,string"`
	MasterSslCert             sql.NullString `json:"master_ssl_cert" db:"Master_SSL_Cert"swaggertype:"primitive,string"`
	MasterSslCipher           sql.NullString `json:"master_ssl_cipher" db:"Master_SSL_Cipher"swaggertype:"primitive,string"`
	MasterSslKey              sql.NullString `json:"master_ssl_key" db:"Master_SSL_Key"swaggertype:"primitive,string"`
	SecondsBehindMaster       int            `json:"seconds_behind_master" db:"Seconds_Behind_Master"`
	MasterSslVerifyServerCert string         `json:"master_ssl_verify_server_cert" db:"Master_SSL_Verify_Server_Cert"`
	LastIoErrno               int            `json:"last_io_errno" db:"Last_IO_Errno"`
	LastIoError               sql.NullString `json:"last_io_error" db:"Last_IO_Error"swaggertype:"primitive,string"`
	LastSqlErrno              int            `json:"last_sql_errno" db:"Last_SQL_Errno"`
	LastSqlError              sql.NullString `json:"last_sql_error" db:"Last_SQL_Error"swaggertype:"primitive,string"`
	ReplicateIgnoreServerIds  interface{}    `json:"replicate_ignore_server_ids" db:"Replicate_Ignore_Server_Ids"`
	MasterServerId            int            `json:"master_server_id" db:"Master_Server_Id"`
	MasterUuid                uuid.UUID      `json:"master_uuid" db:"Master_UUID"`
	MasterInfoFile            string         `json:"master_info_file" db:"Master_Info_File"`
	SqlDelay                  int            `json:"sql_delay" db:"SQL_Delay"`
	SqlRemainingDelay         sql.NullInt64  `json:"sql_remaining_delay" db:"SQL_Remaining_Delay"`
	SlaveSqlRunningState      string         `json:"slave_sql_running_state" db:"Slave_SQL_Running_State"`
	SlaveStatus               string         `json:"slave_status" db:"Slave_Status"`
	MasterRetryCount          int            `json:"master_retry_count" db:"Master_Retry_Count"`
	MasterBind                sql.NullString `json:"master_bind" db:"Master_Bind"swaggertype:"primitive,string"`
	LastIoErrorTimestamp      sql.NullString `json:"last_io_error_timestamp" db:"Last_IO_Error_Timestamp"swaggertype:"primitive,string"`
	LastSqlErrorTimestamp     sql.NullString `json:"last_sql_error_timestamp" db:"Last_SQL_Error_Timestamp"swaggertype:"primitive,string"`
	MasterSslCrl              sql.NullString `json:"master_ssl_crl" db:"Master_SSL_Crl"swaggertype:"primitive,string"`
	MasterSslCrlpath          sql.NullString `json:"master_ssl_crlpath" db:"Master_SSL_Crlpath"swaggertype:"primitive,string"`
	RetrievedGtidSet          sql.NullString `json:"retrieved_gtid_set" db:"Retrieved_Gtid_Set"swaggertype:"primitive,string"`
	ExecutedGtidSet           sql.NullString `json:"executed_gtid_set" db:"Executed_Gtid_Set"swaggertype:"primitive,string"`
	AutoPosition              int            `json:"auto_position" db:"Auto_Position"`
	ReplicateRewritedb        sql.NullString `json:"replicate_rewrite_db" db:"Replicate_Rewrite_DB"swaggertype:"primitive,string"`
	ChannelName               sql.NullString `json:"channel_name" db:"Channel_Name"swaggertype:"primitive,string"`
	MasterTlsVersion          sql.NullString `json:"master_tls_version" db:"Master_TLS_Version"swaggertype:"primitive,string"`
}

func (s *SlaveStatus) MarshalJSON() ([]byte, error) {
	type Alias SlaveStatus
	type u struct {
		ReplicateIgnoreServerIds  driver.Value `json:"replicate_ignore_server_ids"`
		ReplicateDodb             driver.Value `json:"replicate_do_db"`
		ReplicateIgnoredb         driver.Value `json:"replicate_ignore_db"`
		ReplicateDoTable          driver.Value `json:"replicate_do_table"`
		ReplicateIgnoreTable      driver.Value `json:"replicate_ignore_table"`
		ReplicateWildDoTable      driver.Value `json:"replicate_wild_do_table"`
		ReplicateWildIgnoreTable  driver.Value `json:"replicate_wild_ignore_table"`
		LastError                 driver.Value `json:"last_error"`
		MasterSslCaFile           driver.Value `json:"master_ssl_ca_file"`
		MasterSslCaPath           driver.Value `json:"master_ssl_ca_path"`
		MasterSslCert             driver.Value `json:"master_ssl_cert"`
		MasterSslCipher           driver.Value `json:"master_ssl_cipher"`
		MasterSslKey              driver.Value `json:"master_ssl_key"`
		LastIoError               driver.Value `json:"last_io_error"`
		LastSqlError              driver.Value `json:"last_sql_error"`
		MasterBind                driver.Value `json:"master_bind"`
		LastIoErrorTimestamp      driver.Value `json:"last_io_error_timestamp"`
		LastSqlErrorTimestamp     driver.Value `json:"last_sql_error_timestamp"`
		MasterSslCrl              driver.Value `json:"master_ssl_crl"`
		MasterSslCrlpath          driver.Value `json:"master_ssl_crlpath"`
		RetrievedGtidSet          driver.Value `json:"retrieved_gtid_set"`
		ExecutedGtidSet           driver.Value `json:"executed_gtid_set"`
		ReplicateRewritedb        driver.Value `json:"replicate_rewrite_db"`
		ChannelName               driver.Value `json:"channel_name"`
		MasterTlsVersion          driver.Value `json:"master_tls_version"`
		*Alias
	}

	str := u{Alias: (*Alias)(s)}
	str.ReplicateDodb, _ = s.ReplicateDodb.Value()
	str.ReplicateIgnoredb, _ = s.ReplicateIgnoredb.Value()
	str.ReplicateDoTable, _ = s.ReplicateDoTable.Value()
	str.ReplicateIgnoreTable, _ = s.ReplicateIgnoreTable.Value()
	str.ReplicateWildDoTable, _ = s.ReplicateWildDoTable.Value()
	str.ReplicateWildIgnoreTable, _ = s.ReplicateWildIgnoreTable.Value()
	str.LastError, _ = s.LastError.Value()
	str.MasterSslCaFile, _ = s.MasterSslCaFile.Value()
	str.MasterSslCaPath, _ = s.MasterSslCaPath.Value()
	str.MasterSslCert, _ = s.MasterSslCert.Value()
	str.MasterSslCipher, _ = s.MasterSslCipher.Value()
	str.MasterSslKey, _ = s.MasterSslKey.Value()
	str.LastIoError, _ = s.LastIoError.Value()
	str.LastSqlError, _ = s.LastSqlError.Value()
	str.MasterBind, _ = s.MasterBind.Value()
	str.LastIoErrorTimestamp, _ = s.LastIoErrorTimestamp.Value()
	str.LastSqlErrorTimestamp, _ = s.LastSqlErrorTimestamp.Value()
	str.MasterSslCrl, _ = s.MasterSslCrl.Value()
	str.MasterSslCrlpath, _ = s.MasterSslCrlpath.Value()
	str.RetrievedGtidSet, _ = s.RetrievedGtidSet.Value()
	str.ExecutedGtidSet, _ = s.ExecutedGtidSet.Value()
	str.ReplicateRewritedb, _ = s.ReplicateRewritedb.Value()
	str.ChannelName, _ = s.ChannelName.Value()
	str.MasterTlsVersion, _ = s.MasterTlsVersion.Value()

	b, err := json.Marshal(str)

	return b, errors.Wrap(err, "could not marshal slave status")
}

type MysqlProcessList struct {
	ItemID  int       `json:"-"db:"itemId"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	ID           uint64         `json:"id" db:"id"`
	User         string         `json:"user" db:"user"`
	Host         string         `json:"host" db:"host"`
	DB           sql.NullString `json:"db" db:"db"swaggertype:"primitive,string"`
	Command      string         `json:"command" db:"command"`
	Time         int64          `json:"time" db:"time"`
	State        sql.NullString `json:"state" db:"state"swaggertype:"primitive,string"`
	Info         sql.NullString `json:"info" db:"info"swaggertype:"primitive,string"`
	RowsSent     uint64         `json:"rows_sent" db:"rows_sent"`
	RowsExamined uint64         `json:"rows_examined" db:"rows_examined"`
}

func (m *MysqlProcessList) MarshalJSON() ([]byte, error) {
	type Alias MysqlProcessList
	type u struct {
		DB           driver.Value `json:"db"`
		State        driver.Value `json:"state"`
		Info         driver.Value `json:"info"`
		*Alias
	}

	str := u{
		Alias: (*Alias)(m),
	}

	str.DB, _ = m.DB.Value()
	str.State, _ = m.State.Value()
	str.Info, _ = m.Info.Value()

	b, err := json.Marshal(&str)

	return b, errors.Wrap(err, "could not marshal mysql processlist")
}

type EngineINNODBStatus struct {
	ID      int       `db:"id" json:"-"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	Name   string `db:"name" json:"name"`
	Type   string `db:"type" json:"type"`
	Status string `db:"status" json:"status"`
}

type Thread struct {
	ID      int       `db:"id"json:"-"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	ThreadID           int64          `json:"thread_id" db:"thread_id"`
	Name               string         `json:"name" db:"name"`
	Type               string         `json:"type" db:"type"`
	ProcessListID      sql.NullInt64  `json:"process_list_id" db:"processlist_id"swaggertype:"primitive,integer"`
	ProcessListUser    sql.NullString `json:"process_list_user" db:"processlist_user"swaggertype:"primitive,string"`
	ProcessListHost    sql.NullString `json:"process_list_host" db:"processlist_host"swaggertype:"primitive,string"`
	ProcessListDB      sql.NullString `json:"process_list_db" db:"processlist_db"swaggertype:"primitive,string"`
	ProcessListCommand sql.NullString `json:"process_list_command" db:"processlist_command"swaggertype:"primitive,string"`
	ProcessListTime    sql.NullInt64  `json:"process_list_time" db:"processlist_time"swaggertype:"primitive,string"`
	ProcessListState   sql.NullString `json:"process_list_state" db:"processlist_state"swaggertype:"primitive,string"`
	ProcessListInfo    sql.NullString `json:"process_list_info" db:"processlist_info"swaggertype:"primitive,string"`
	ParentThreadID     sql.NullInt64  `json:"parent_thread_id" db:"parent_thread_id"swaggertype:"primitive,integer"`
	Role               sql.NullString `json:"role" db:"role"swaggertype:"primitive,string"`
	Instrumented       string         `json:"instrumented" db:"instrumented"`
	History            string         `json:"history" db:"history"`
	ConnectionType     sql.NullString `json:"connection_type" db:"connection_type"swaggertype:"primitive,integer"`
	ThreadOsID         sql.NullInt64  `json:"thread_os_id" db:"thread_os_id"swaggertype:"primitive,integer"`
}

func (t *Thread) MarshalJSON() ([]byte, error) {
	type Alias Thread
	type u struct {
		ProcessListID      driver.Value `json:"process_list_id"`
		ProcessListUser    driver.Value `json:"process_list_user"`
		ProcessListHost    driver.Value `json:"process_list_host"`
		ProcessListDB      driver.Value `json:"process_list_db"`
		ProcessListCommand driver.Value `json:"process_list_command"`
		ProcessListTime    driver.Value `json:"process_list_time"`
		ProcessListState   driver.Value `json:"process_list_state"`
		ProcessListInfo    driver.Value `json:"process_list_info"`
		ParentThreadID     driver.Value `json:"parent_thread_id"`
		Role               driver.Value `json:"role"`
		ConnectionType     driver.Value `json:"connection_type"`
		ThreadOsID         driver.Value `json:"thread_os_id"`
		*Alias
	}

	str := u{Alias: (*Alias)(t)}
	str.ProcessListID, _ = t.ProcessListID.Value()
	str.ProcessListUser, _ = t.ProcessListUser.Value()
	str.ProcessListHost, _ = t.ProcessListHost.Value()
	str.ProcessListDB, _ = t.ProcessListDB.Value()
	str.ProcessListCommand, _ = t.ProcessListCommand.Value()
	str.ProcessListTime, _ = t.ProcessListTime.Value()
	str.ProcessListState, _ = t.ProcessListState.Value()
	str.ProcessListInfo, _ = t.ProcessListInfo.Value()
	str.ParentThreadID, _ = t.ParentThreadID.Value()
	str.Role, _ = t.Role.Value()
	str.ConnectionType, _ = t.ConnectionType.Value()
	str.ThreadOsID, _ = t.ThreadOsID.Value()

	b, err := json.Marshal(&str)

	return b, errors.Wrap(err, "could not marshall ps thread")
}

type UnixProcessList struct {
	ID      int       `db:"id"json:"-"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	Text string `db:"text"json:"text"`
}

type Top struct {
	ID      int       `db:"id"json:"-"`
	CycleID uuid.UUID `db:"cycle_id"json:"-"`

	Text string `db:"text"json:"text"`
}
