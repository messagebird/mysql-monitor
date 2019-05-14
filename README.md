# Mysql Monitor

This tool is basically Atop with some mysql queries included such as:
- process list 
- threads
- engine innodb status
- slave status
- (more can be added if needed)

Which can be either saved inside a slqite3 db(experimental)
(Some features here are still missing, e.g export from sqite to file). Or to .json.gz/.txt.gz files on disk


## Saving .gz's to disk

To use this mode, pass the `--not-in-sqlite` parameter.

When the monitor is ran to save .gz's to disk (inside the /tmp/no-sql folder), you can expect the following file structure:
```
drwxr-xr-x    - ogkevin 25 Apr 11:18 tmp/no-sql
drwxr-xr-x    - ogkevin 16 Apr 11:08 ├── mysql
drwxr-xr-x    - ogkevin 29 Apr 12:04 │  ├── engine-innodb
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │  └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │     └── 1000
.rw-r--r-- 1.4k ogkevin 29 Apr 12:05 │  │        ├── 2019-04-29T100550Z.txt.gz
.rw-r--r-- 1.4k ogkevin 29 Apr 12:05 │  │        └── 2019-04-29T100600Z.txt.gz
drwxr-xr-x    - ogkevin 29 Apr 12:04 │  ├── process-list
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │  └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │     └── 1000
.rw-r--r--  249 ogkevin 29 Apr 12:05 │  │        ├── 2019-04-29T100550Z.json.gz
.rw-r--r--  281 ogkevin 29 Apr 12:05 │  │        └── 2019-04-29T100600Z.json.gz
drwxr-xr-x    - ogkevin 29 Apr 12:04 │  ├── ps_threads
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │  └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:05 │  │     └── 1000
.rw-r--r--  868 ogkevin 29 Apr 12:05 │  │        ├── 2019-04-29T100550Z.json.gz
.rw-r--r--  834 ogkevin 29 Apr 12:05 │  │        └── 2019-04-29T100600Z.json.gz
drwxr-xr-x    - ogkevin 29 Apr 12:04 │  └── slave-status
drwxr-xr-x    - ogkevin 29 Apr 12:05 │     └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:05 │        └── 1000
.rw-r--r--   26 ogkevin 29 Apr 12:05 │           ├── 2019-04-29T100550Z.json.gz
.rw-r--r--   26 ogkevin 29 Apr 12:05 │           └── 2019-04-29T100600Z.json.gz
drwxr-xr-x    - ogkevin 25 Apr 11:06 └── system
drwxr-xr-x    - ogkevin 29 Apr 12:04    ├── ps
drwxr-xr-x    - ogkevin 29 Apr 12:05    │  └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:05    │     └── 1000
.rw-r--r--  294 ogkevin 29 Apr 12:05    │        ├── 2019-04-29T100550Z.txt.gz
.rw-r--r--  291 ogkevin 29 Apr 12:05    │        └── 2019-04-29T100600Z.txt.gz
drwxr-xr-x    - ogkevin 29 Apr 12:04    └── top
drwxr-xr-x    - ogkevin 29 Apr 12:03       └── 19-04-29
drwxr-xr-x    - ogkevin 29 Apr 12:06          └── 1000
.rw-r--r--  422 ogkevin 29 Apr 12:05             ├── 2019-04-29T100550Z.txt.gz
.rw-r--r--  421 ogkevin 29 Apr 12:06             └── 2019-04-29T100600Z.txt.gz
```

This is the file output by running the monitor with a 10 sec interval for 2 cycles.

You will get 2 main dirs which are `mysql` and `system`. Which contains mysql and system logs respectively.
Each dir will then be split in dirs for each log type. These dirs will then be split by date. The date dirs will then be
split by the hour and inside the hour dirs the actual logs files will be located. This should make the quest to find
logs of a specific time and date easier. 

## Saving to sqlite
 
*This mode uses more disk space*

The idea behind this mode is that it will store everything to sqlite and exposes this via an api. This way you could
build on top of this api. e.g. a dashboard containing all logs of all mysql servers that you're running.

The API spec can be found at [swagger.json](./docs/swagger.json) or by
visiting the `/swagger` endpoint on `localhost:port`

What this api can do:
- List all cycles
- Get all logs of each type per cycle id.

# Commands and args
## Monitor command (m)
|Argument|Description|
|---|---|
|--interval| How often the monitor should collect logs.|
|--log-retention| How long these logs should be saved. The monitor will delete files that are eligible for deletion every 60 sec|
|--mysql-credential-path|The path containing the credentials to connect to the mysql server. For more info on this see [mysql cred section](#mysql-credential-file)|
|--output-dir|The path where the sqlite db/log files containing the data should be saved (default: "./mysql-monitor-db.sqlite")|
|--exclude-\<type>|Exclude \<type> from being logged.|
|--not-in-sqlite|Use logging to disk instead of sqlite see [Saving .gz's to disk](#saving-.gz's-to-disk)|

## Get command (g)
This command is only needed if using the "save in sqlite" mode.

This command allows one to "query" the logs from the sqlite db. 

The syntax is `mysql-monitor get <category> <type> <cycle-id>`

To get a cycle id run: `mysql-monitor get cycles`

# Mysql credential file
The contents of the mysql credential file should contain:

```
[client]
user=root
password=root
host=127.0.0.1
```

Connecting via socket is also supported.
