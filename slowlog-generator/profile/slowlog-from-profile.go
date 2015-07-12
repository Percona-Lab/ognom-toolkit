/*
This program reads a mongodb/tokumx system.profile collection and generates a MySQL formatted slow query log, that can then be used with pt-query-digest for workload analysis
*/
package main

import (
	"flag"
	"fmt"
	"os"

	"../util"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Command line args are pretty self-explanatory: We only need a mongo url (to be used with mgo.Dial) and a database name. We default to 127.0.0.1/local
var MONGO = flag.String("mongo", "127.0.0.1", "The mongod/mongos instance to connect to")
var DB = flag.String("db", "local", "The database that has the system.profile collection we need to process")

/*

Sample slow query log entry header from MySQL

# Time: 150402 14:02:44
# User@Host: [fernandoipar] @ localhost []
# Thread_id: 13  Schema:   Last_errno: 0  Killed: 0
# Query_time: 0.000052  Lock_time: 0.000000  Rows_sent: 1  Rows_examined: 0  Rows_affected: 0  Rows_read: 0
# Bytes_sent: 90
SET timestamp=1427994164;
db.sample.find({a:"test", b:"another test"});

Header from Percona Server with log_slow_verbosity set to all:

# User@Host: [fernandoipar] @ localhost []
# Thread_id: 2  Schema:   Last_errno: 0  Killed: 0
# Query_time: 0.000003  Lock_time: 0.000000  Rows_sent: 0  Rows_examined: 0  Rows_affected: 0  Rows_read: 0
# Bytes_sent: 0  Tmp_tables: 0  Tmp_disk_tables: 0  Tmp_table_sizes: 0
# QC_Hit: No  Full_scan: No  Full_join: No  Tmp_table: No  Tmp_table_on_disk: No
# Filesort: No  Filesort_on_disk: No  Merge_passes: 0
# No InnoDB statistics available for this query
SET timestamp=1435605887;
# administrator command: Quit;

*/

func main() {
	flag.Parse()
	if flag.NFlag() == 0 {
		fmt.Fprintf(os.Stderr, "Running with default flags. mongo=%v, db=%v\n", *MONGO, *DB)
	}
	session, err := mgo.Dial(*MONGO)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	col := session.DB(*DB).C("system.profile")

	var results []map[string]interface{}
	err = col.Find(bson.M{}).All(&results)

	if err != nil {
		panic(err)
	}

	for _, v := range results {
		var info util.OpInfo = make(util.OpInfo)
		_, _query, info := util.RecurseJsonMap(v)
		query := ""
		if v, ok := info["op"]; ok {
			ns := info["ns"] // ns is always there or we must just crash/behave erratically
			switch v {
			case "query":
				limit := info["ntoreturn"]
				skip := info["ntoskip"]
				if limit == "0" {
					limit = ""
				} else {
					limit = fmt.Sprintf(".limit(%v)", limit)
				}
				if skip == "0" {
					skip = ""
				} else {
					skip = fmt.Sprintf(".skip(%v)", skip)
				}
				query = fmt.Sprintf("%v.find{%v}%v%v;", ns, _query, skip, limit)
			case "insert":
				query = fmt.Sprintf("%v.insert{%v};", ns, _query)
			case "update":
				query = fmt.Sprintf("%v.update({%v},{%v});", ns, _query, info["updateobj"])
			case "remove":
				query = fmt.Sprintf("%v.remove({%v});", ns, _query)
			case "getmore":
				query = fmt.Sprintf("%v.getmore;", ns)
			case "command":
				query = fmt.Sprintf("%v({%v});", ns, info["command"])
			default:
				query = fmt.Sprintf("__UNIMPLEMENTED__ {%v};", _query)
			}
		}
		fmt.Print(util.GetSlowQueryLogHeader(info), query, "\n")
	}

}
