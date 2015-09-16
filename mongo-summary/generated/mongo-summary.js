
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
      var result = {};
      var aux;
      aux = db.hostInfo()["system"]["currentTime"];
      result["serverTime"] = aux.getFullYear() + "-" + getFilledDatePart(aux.getMonth()) + "-" + getFilledDatePart(aux.getDay()) + " " + aux.toTimeString();
      aux = db.currentOp()["inprog"];
      result["inprog"] = aux.length + " operations in progress";
      result["hostname"] = db.hostInfo()["system"]["hostname"];
      result["serverStatus"] = db.serverStatus();
      return result;
  }

  function getReplicationSummary(db) {
      var result = {};
      var rstatus = db._adminCommand("replSetGetStatus");
      result["ok"] = rstatus["ok"];
      if (rstatus["ok"]==0) {
          // This is either not a replica set, or there is an error
          if (rstatus["errmsg"] == "not running with --replSet") {
             result["summary"] = "Standalone mongod" 
          } else {
              result["summary"] = "Replication error: " + rstatus["errmsg"]
          }
      } else {
          // This is a replica set
          var secondaries = 0;
          var arbiters = 0;
          result["members"] = [];
          rstatus["members"].forEach(
              function (element, index, array) {
                  if (element["self"]) {
                      result["summary"] = "Node is " + element["stateStr"] + " in a " + rstatus["members"].length + " members replica set"
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
  var aux = getInstanceBasicInfo(db);
  print("Report generated on " + aux["hostname"] + " at " + aux["serverTime"]);
  print(aux["inprog"]);
  if (isMongos()) {
      print(getHeader("Sharding Summary (mongos detected)",FILLER,LENGTH));
      aux = getShardingSummary();
      print("Detected " + aux["shards"].length + " shards");
      print("Sharded databases: ");
      aux["shardedDatabases"].forEach(function (element, array, index) {print("  " + element["_id"]);});
      print("");
      print("Unsharded databases: ");
      aux["unshardedDatabases"].forEach(function (element, array, index) {print("  " + element["_id"]);});
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
      aux = getReplicationSummary(db);
      print(aux["summary"]);
      print(aux["summaryExtra"]);
      if (aux["members"].length > 0) {
          print(getHeader("Replica set members",FILLER,LENGTH));
          aux["members"].forEach(
              function(member, array, index) {
                  print(member);
              }
          );
      }
  } 
  printjson(aux["serverStatus"]);
