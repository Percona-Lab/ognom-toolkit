/*
This is meant to be used on MongoDB/TokuMX versions < 2.6, which is when cleanupOrphaned was introduced (http://docs.mongodb.org/manual/reference/command/cleanupOrphaned/)
You just need to set the database and collection variables to the right values and then run the script. It will iterate over every document in the collection, and manually look for it on every shard, so it can take a while to run (it has known to take days on some cases).
*/

var database = "sample";
var collection = "tests";
// if you know (from discrepancy between db.collection.find().count() and counting through a cursor) how many orphaned documents you have, 
// you can set the value here, and the script will throw an exception and exit when this count is reached
var exitAt = -1;

// No customization needed below this line

// We will store a connection to every shard here. 
var connections = [];

var found = 0;

config = db.getMongo().getDB("config");
config.shards.find().forEach(
    function (shard,_a,_i) {
        connections.push(new Mongo(shard["host"]));
    }
);

db.getMongo().getDB(database).getCollection(collection).find().forEach(
    function (doc, _a, _i) {
        count = 0;
        onshards = [];
        connections.forEach(
            function(con, _a, _i) {
	  each_count = con.getDB(database).getCollection(collection).find(doc).count()
	  if (each_count >= 1) {
                    count += each_count;
                    onshards.push(con);
                }
            }
);
        if (count > 1) {
	    found += 1;
            print("duplicate doc: "+doc['_id']+" found on: ");
            onshards.forEach(function(shard,_a,_i) {print(shard.toString())});
	    if (exitAt > 0 && exitAt >= found) {
		throw "Found " + exitAt + " matches, exiting";
	    }
        }
    }
);
