
function getFilledDatePart(datepart) {
    return datepart < 10 ? "0" + datepart : datepart;
}

function getHeader(header, filler, length) {
    var result = "\n# " + header + " ";
    if (result.length < length) {
        for (i=result.length; i<length; i++) {
            result += filler
        }
    }
    return result + "\n";
}
var LENGTH = 62;
var FILLER = "#";
function isMongos() {
    return db.runCommand({isMaster: 1})["msg"] == "isdbgrid";
}

function getInstanceBasicInfo(db) {
    var result = {}
    var aux
    aux = db.hostInfo()["system"]["currentTime"]
    result["serverTime"] = aux.getFullYear() + "-" + getFilledDatePart(aux.getMonth()) + "-" + getFilledDatePart(aux.getDay()) + " " + aux.toTimeString()
    aux = db.currentOp()["inprog"]
    result["inprog"] = aux.length + " operations in progress"
    result["hostname"] = db.hostInfo()["system"]["hostname"]
    result["serverStatus"] = db._adminCommand({serverStatus:1})
    result["parameters"] = db._adminCommand({getParameter:'*'})
    result["cmdLineOpts"] = db._adminCommand({getCmdLineOpts:1})
    return result
}

function getReplicationSummary(db) {
    var result = {};
    var rstatus = db._adminCommand("replSetGetStatus");
    result["ok"] = rstatus["ok"];
    if (rstatus["ok"]==0) {
        // This is either not a replica set, or there is an error
        if (rstatus["errmsg"] == "not running with --replSet") {
           result["summary"] = "Standalone mongod" 
           result["summaryExtra"] = ""
        } else {
            result["summary"] = "Replication error: " + rstatus["errmsg"]
            result["summaryExtra"] = ""
        }
        result["members"] = [];
    } else {
        // This is a replica set
        var secondaries = 0;
        var arbiters = 0;
  result["summary"] = "This is a replica set but I could not figure out this node's role"
        result["members"] = [];
        rstatus["members"].forEach(
            function (element, index, array) {
                if (element["self"]) {
                    result["summary"] = "Node is " + element["stateStr"] + " in a " + rstatus["members"].length + " members replica set"
	      if (!result["summary"]) {
		  result["summary"] = "This is a replica set, but something went wrong when trying to figure out this node's role"
	      }
                } else {
                    if (element["state"] == 2) {
                        secondaries++;
                    } else if (element["state"] == 7) {
                        arbiters++;
                    }
                }
                result["members"].push(element["name"]);
            }
        )
        result["summaryExtra"] = "The set has " + secondaries + " secondaries and " + arbiters + " arbiters";
    }
    return result;
} 

function getShardingSummary() {
    var result = {};
    result["shards"] = [];
    result["shardedDatabases"] = [];
    result["unshardedDatabases"] = [];
    var con = db.getMongo().getDB("config");
    con.databases.find().forEach(
        function (element, index, array) {
            if (element["partitioned"]) {
                result["shardedDatabases"].push(element);
            } else {
                result["unshardedDatabases"].push(element);
            }
        }
    );
    con.shards.find().forEach (
        function (element, index, array) {
            result["shards"].push({_id: element["_id"], host: element["host"].slice(element["host"].indexOf("/")+1,element["host"].length)});
        }
    );
    return result;
}

function getShardsInfo() {
    var shardingSummary = getShardingSummary();
    var result = {};
    result["shards"] = [];
    shardingSummary["shards"].forEach(
        function (element, index, array) {
            element["host"].split(",").forEach(
                function (element, index, array) {
                    var db = new Mongo(element).getDB("local")
                    result["shards"].push({
                    host: element,
                    hostInfo: getInstanceBasicInfo(db),
                    replicationSummary: getReplicationSummary(db)
                    })
                }
            ) 
        }
    );
    return result;
}

print(getHeader("Percona Toolkit MongoDB Summary Report",FILLER,LENGTH));
var basicInfo = getInstanceBasicInfo(db);
print("Report generated on " + basicInfo["hostname"] + " at " + basicInfo["serverTime"]);
print(basicInfo["inprog"]);
if (isMongos()) {
    print(getHeader("Sharding Summary (mongos detected)",FILLER,LENGTH));
    shardsInfo = getShardingSummary();
    print("Detected " + shardsInfo["shards"].length + " shards");
    print("Sharded databases: ");
    shardsInfo["shardedDatabases"].forEach(function (element, array, index) {print("  " + element["_id"]);});
    print("");
    print("Unsharded databases: ");
    shardsInfo["unshardedDatabases"].forEach(function (element, array, index) {print("  " + element["_id"]);});
    print("");
    print(getHeader("Shards detail",FILLER,LENGTH));
    getShardsInfo()["shards"].forEach(
        function (element, array, index) {
            print("Shard " + element["_id"] + " @ " + element["host"]);
            print("(" + element["hostInfo"]["inprog"] + ")");
            print(element["replicationSummary"]["summary"]);
            print(element["replicationSummary"]["summaryExtra"]);
            print("");
        }
    );
} else { 
    print(getHeader("Replication summary",FILLER,LENGTH));
    replicationSummary = getReplicationSummary(db);
    if (replicationSummary["summary"]) {
  print(replicationSummary["summary"])
    } else {
  print("Something is wrong with the replication summary (it is undefined)")
    }
    if (replicationSummary["summaryExtra"]) {
  print(replicationSummary["summaryExtra"]);
    }
    if (replicationSummary["members"].length > 0) {
        print(getHeader("Replica set members",FILLER,LENGTH));
        replicationSummary["members"].forEach(
            function(member, array, index) {
                print(member);
            }
        );
    }
} 
print(getHeader("Server Status",FILLER,LENGTH))
printjson(basicInfo["serverStatus"]);
print(getHeader("Server Parameters",FILLER,LENGTH))
printjson(basicInfo["parameters"]);
print(getHeader("Command Line Options",FILLER,LENGTH))
printjson(basicInfo["cmdLineOpts"]);
