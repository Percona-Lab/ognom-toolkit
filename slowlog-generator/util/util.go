/*
Internal package that provides utility functions for the tools that process the mongodb profiler collection or that capture mongodb network traffic.
*/
package util

import (
	"fmt"
	"time"

	"gopkg.in/mgo.v2/bson"
)

/*
This is used to save data points that will be used when generating the slow query log entry header.
Examples include response time, lock time, user, host, etc.
*/
type OpInfo map[string]string

/*
This walks an array and generates a string representation of its contents. It recurses down substructures as needed, provided they're one of
a [string]interface{} map (i.e. a json doc) or another array.
There are no depth controls, so a sufficiently deep structure could blow the stack
*/
func RecurseArray(input []interface{}) (output string) {
	output = "["
	i := 0
	for k, v := range input {
		i++
		comma := ", "
		if i == len(input) {
			comma = ""
		}
		switch extracted_v := v.(type) {
		case map[string]interface{}:
			aux, _, _ := RecurseJsonMap(extracted_v)
			output += fmt.Sprintf("%v:{%v}%v", k, aux, comma)
		case []interface{}:
			output += fmt.Sprintf("%v:%v%v", k, RecurseArray(extracted_v), comma)
		default:
			output += fmt.Sprintf("%v:%v%v", k, extracted_v, comma)
		}
	}
	output += fmt.Sprintf("]")
	return output
}

/*
This does the same as the previous function, but with a json document, and populating info on the way.
The same stack disclaimer applies here.
This is the point where data types are converted to their appropriate representation. Anything above this call
will deal with strings only.
*/
func RecurseJsonMap(json map[string]interface{}) (output string, query string, info OpInfo) {
	i := 0
	info = make(OpInfo)
	for k, v := range json {
		if k == "user" || k == "ns" || k == "millis" || k == "responseLength" || k == "client" || k == "nscanned" || k == "ntoreturn" || k == "ntoskip" || k == "nreturned" || k == "op" || k == "ninserted" || k == "ndeleted" || k == "nModified" || k == "cursorid" {
			info[k] = fmt.Sprint(v)
		}
		if k == "query" {
			query, _, _ = RecurseJsonMap(v.(map[string]interface{}))
		}
		if k == "updateobj" {
			updateobj, _, _ := RecurseJsonMap(v.(map[string]interface{}))
			info[k] = updateobj
		}
		if k == "command" {
			command, _, _ := RecurseJsonMap(v.(map[string]interface{}))
			info[k] = command
		}
		i++
		comma := ", "
		if i == len(json) {
			comma = ""
		}
		switch extracted_v := v.(type) {
		case string, time.Time, int, int32, int64:
			output += fmt.Sprintf("%v:%v%v", k, extracted_v, comma)
		case float64:
			output += fmt.Sprintf("%v:%v%v", k, float64(extracted_v), comma)
		case map[string]interface{}:
			auxstr, _query, auxOpInfo := RecurseJsonMap(extracted_v)
			if _query != "" {
				query = _query
			}
			info = mergeOpInfoMaps(info, auxOpInfo)
			output += fmt.Sprintf("%v:{%v}%v", k, auxstr, comma)
		case []interface{}:
			output += fmt.Sprintf("%v:%v%v", k, RecurseArray(extracted_v), comma)
		case bson.ObjectId:
			output += fmt.Sprintf("%v:%v%v", k, extracted_v.String(), comma)
		default:
			output += fmt.Sprintf("%v:%T%v", k, extracted_v, comma)
		}

	}
	return output, query, info
}

/*
This merges OpInfo maps, as we may create more than one while recursing down a document.
It is a very primite merge because, in theory, there should be no colliding key names in the fields
we're collecting.
*/
func mergeOpInfoMaps(s1 OpInfo, s2 OpInfo) (result OpInfo) {
	result = make(OpInfo)
	for k, v := range s1 {
		if v2, ok := s2[v]; ok {
			result[k] = fmt.Sprintf("%v | %v", v, v2)
		} else {
			result[k] = v
		}
	}
	return result
}

// This is just a helper function to not pollute the header generator with default initialazers
func initSlowQueryLogHeaderVars(input OpInfo) (output OpInfo) {
	output = make(OpInfo)
	output["millis"] = "n/a"
	output["sent"] = "n/a"
	output["user"] = ""
	output["host"] = ""
	output["inserted"] = "0"
	output["scanned"] = "0"
	output["deleted"] = "0"
	output["returned"] = "0"
	if v, ok := input["millis"]; ok {
		output["millis"] = v
	}
	if v, ok := input["sent"]; ok {
		output["sent"] = v
	}
	if v, ok := input["user"]; ok {
		output["user"] = v
	}
	if v, ok := input["client"]; ok {
		output["host"] = v
	}
	if v, ok := input["ninserted"]; ok {
		output["inserted"] = v
	}
	if v, ok := input["nscanned"]; ok {
		output["scanned"] = v
	}
	if v, ok := input["ndeleted"]; ok {
		output["deleted"] = v
	}
	if v, ok := input["nreturned"]; ok {
		output["returned"] = v
	}
	return output
}

/*
This formats a slow query log entry header, filling in performance information from the OpInfo map
*/
func GetSlowQueryLogHeader(input OpInfo) (output string) {

	info := initSlowQueryLogHeaderVars(input)
	affected := info["inserted"]
	if input["op"] == "remove" {
		affected = info["deleted"]
	}
	now := time.Now().Format("060102 15:04:05")
	output = fmt.Sprintf("# Time: %v\n", now)
	output += fmt.Sprintf("# User@Host: %v @ %v []\n", info["user"], info["host"])
	output += "# Thread_id: 1 Schema: Last_errno: 0 Killed: 0\n"
	output += fmt.Sprintf("# Query_time: %v Lock_time: 0 Rows_sent: %v Rows_examined: %v Rows_affected: %v Rows_read: 1\n", info["millis"], info["returned"], info["scanned"], affected)
	output += fmt.Sprintf("# Bytes_sent: %v\n", info["sent"])
	output += fmt.Sprintf("SET timestamp=%v;\n", time.Now().Unix())
	return output
}
