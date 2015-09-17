/*

Use this to look for redundant indexes (based on one index being a leftmost prefix of another one) on all collections of a given mongod/mongos instance. 
Sample output: 

telecaster:utils fernandoipar$ mongo find-duplicate-indexes.js
MongoDB shell version: 3.0.4
connecting to: test
test.test.seq_1 is a prefix of test.test.seq_1_ts_1
test.test.ts_1 is a prefix of test.test.ts_1_seq_1

*/


// This object will hold all the index definitions on the system
indexes = {}

// Iterate over each collection of each database and save the result of getIndexes() on the indexes object
db.adminCommand('listDatabases')["databases"].forEach(
    function (element, array, index) {
	var auxdb = db.getSiblingDB(element["name"])
	var cols = auxdb.getCollectionNames()
	indexes[element["name"]] = {}
	auxdb.getCollectionNames().forEach(function(collection) {
	    indexes[element["name"]][collection] = auxdb[collection].getIndexes()
	})
})

// Helper function to generate a string representation of an index. 
// This representation is just a concatenation of the fields of an index, so that we can then
// use String.startsWith to look for redundant indexes. 
function index_to_str(index) {
    result = ""
    for (var k in index) {
	result += k
    }
    return result
}

// This aux object has a string representation for every index on the system
indexes_in_db = {}

for (var db in indexes) {
    for (var collection in indexes[db]) {
	indexes[db][collection].forEach(
	    function(index) {
		indexes_in_db[db+"."+collection+"."+index["name"]] = index_to_str(index["key"])
	    })
    }
}

// We can now look for redundant indexes, based on the simple logic that
// if a and b are distinct and a is a substring of b, then a is made redundant by b. 
for (var k in indexes_in_db) {
    aux = indexes_in_db[k]
    for (var k2 in indexes_in_db) {
	aux2 = indexes_in_db[k2]
	if (aux != aux2 && aux.startsWith(aux2)) {
	    print(k2+" is a prefix of "+k)
	}
    }
}


