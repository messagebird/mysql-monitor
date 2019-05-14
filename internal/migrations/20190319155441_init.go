package migration

import (
	"database/sql"
	"github.com/pkg/errors"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20190319155441, Down20190319155441)
}

func Up20190319155441(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`
create table cycle
(
	id text not null
		constraint cycle_pk
			primary key,
	cycle integer not null,
	ran_at timestamp not null,
	took text not null
);

create unique index cylce_ran_at_uindex
	on cycle (ran_at);

create table slave_status
(
	id                            integer      not null
		constraint slave_status_pk
			primary key autoincrement,
	cycle_id text not null,
	Master_Log_File               text         not null,
	Master_Host                   text         not null,
	Master_User                   text         not null,
	Master_Port                   text         not null,
	Connect_Retry                 text         not null,
	Read_Master_Log_Pos           integer      not null,
	Relay_Log_File                text         not null,
	Relay_Log_Pos                 integer      not null,
	Relay_Master_Log_File         text         not null,
	Slave_IO_Running              text         not null,
	Slave_IO_State                text         not null,
	Slave_IO_Status               text         not null,
	Slave_SQL_Running             text         not null,
	Replicate_Do_DB              text         null,
	Replicate_Ignore_DB           text         null,
	Replicate_Do_Table            text         null,
	Replicate_Ignore_Table        text         null,
	Replicate_Wild_Do_Table       text         null,
	Replicate_Wild_Ignore_Table   text         null,
	Last_Errno                    integer,
	Last_Error                    text         not null,
	Skip_Counter                  integer      not null,
	Exec_Master_Log_Pos           integer      not null,
	Relay_Log_Space               integer      not null,
	Until_Condition               text         not null,
	Until_Log_File                text         not null,
	Until_Log_Pos                 integer      not null,
	Master_SSL_Allowed            text         not null,
	Master_SSL_CA_File            text         null,
	Master_SSL_CA_Path            text         null,
	Master_SSL_Cert               text         null,
	Master_SSL_Cipher             text         null,
	Master_SSL_Key                text         null,
	Seconds_Behind_Master         integer      not null,
	Master_SSL_Verify_Server_Cert text         not null,
	Last_IO_Errno                 integer      not null,
	Last_IO_Error                 text         null,
	Last_SQL_Errno                integer      not null,
	Last_SQL_Error                text         null,
	Replicate_Ignore_Server_Ids   integer      not null,
	Master_Server_Id              integer      not null,
	Master_UUID                   text         not null,
	Master_Info_File              text         not null,
	SQL_Delay                     integer      not null,
	SQL_Remaining_Delay           integer      null,
	Slave_SQL_Running_State       text         not null,
	Slave_Status                  text         not null,
	Master_Retry_Count            integer      not null,
	Master_Bind                   text         null,
	Last_IO_Error_Timestamp       text         null,
	Last_SQL_Error_Timestamp      text         null,
	Master_SSL_Crl                text         null,
	Master_SSL_Crlpath            text         null,
	Retrieved_Gtid_Set            text         null,
	Executed_Gtid_Set             text         null,
	Auto_Position                 text integer not null,
	Replicate_Rewrite_DB          text         null,
	Channel_Name                  text         null,
	Master_TLS_Version            text         null,
	foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table mysql_process_list
(
  itemId        integer   not null
    constraint mysql_process_list_pk
      primary key autoincrement,
  cycle_id text not null,
  id            integer   not null,
  user          text      not null,
  host          text      not null,
  db            text      null,
  command       text      not null,
  time          integer   not null,
  state         text      null,
  info          text      null,
  rows_sent     integer   not null,
  rows_examined integer   not null,
  foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table engine_inno_db_status
(
  id         integer   not null
    constraint engine_inno_db_status_pk
      primary key autoincrement,
  cycle_id text not null,
  name       text      not null,
  type       text      not null,
  status     text      not null,
  foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table threads
(
	id integer not null
		constraint threads_pk
			primary key autoincrement,
	cycle_id text not null,
	thread_id integer not null,
	name text not null,
	type text not null,
	processlist_id integer,
	processlist_user text,
	processlist_host text,
	processlist_db text,
	processlist_command text,
	processlist_time integer,
	processlist_state text,
	processlist_info text,
	parent_thread_id integer,
	role text,
	instrumented text not null,
	history text not null,
	connection_type text,
	thread_os_id integer,
	foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table unix_process_list (
  id integer  not null primary key autoincrement,
  cycle_id text not null,
  text text not null,
  foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table top (
  id integer  not null primary key autoincrement,
  cycle_id text not null,
  text text not null,
  foreign key (cycle_id) references cycle(id) on delete cascade 
);

create table file_on_disk (
	cycle_id text not null,
	path text not null,
	foreign key (cycle_id) references cycle(id) on delete cascade 
);

`)
	if err != nil {
		return errors.Wrap(err, "could not apply initial migration")
	}

	return nil
}

func Down20190319155441(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec(`drop table slave_status;
drop table mysql_process_list;
drop table engine_inno_db_status;
drop table threads;
drop table unix_process_list;
drop table top;
drop table file_on_disk;
`)
	if err != nil {
		return errors.Wrap(err, "could not roll back initial migration")
	}
	return nil
}
