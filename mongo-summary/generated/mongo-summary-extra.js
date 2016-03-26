
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
function convertToText(obj) {
    //create an array that will later be joined into a string.
    var string = [];

    //is object
    //    Both arrays and objects seem to return "object"
    //    when typeof(obj) is applied to them. So instead
    //    I am checking to see if they have the property
    //    join, which normal objects don't have but
    //    arrays do.
    if (obj == undefined) {
    	return String(obj);
    } else if (typeof(obj) == "object" && (obj.join == undefined)) {
        for (prop in obj) {
        	if (obj.hasOwnProperty(prop))
            string.push(prop + ": " + convertToText(obj[prop]));
        };
    return "{" + string.join(",") + "}\n";

    //is array
    } else if (typeof(obj) == "object" && !(obj.join == undefined)) {
        for(prop in obj) {
            string.push(convertToText(obj[prop]));
        }
    return "[" + string.join(",") + "]\n";

    //is function
    } else if (typeof(obj) == "function") {
        string.push(obj.toString())

    //all other values can be done with JSON.stringify
    } else {
        string.push(obj)
        //string.push(JSON.stringify(obj))
    }

    return string.join(",");
}

function printExtraDiagnosticsInfo() {
    print(getHeader("Extra info",FILLER,LENGTH));

    db.adminCommand('listDatabases')["databases"].forEach(
        function (element, array, index) {
            var auxdb = db.getSiblingDB(element["name"]);
            var cols = auxdb.getCollectionNames();
            print(element["name"] + " has " + cols.length + " collections and " + element["sizeOnDisk"] + " bytes on disk");
            if (cols.length > 0) {
                print("Collections: ");
                cols.forEach(
                    function (element, array, index) {
                        print("   " + element);
		  auxdb.getCollectionNames().forEach(function(collection) {
		     indexes = auxdb[collection].getIndexes();
		     print("Indexes for " + collection + ":");
		     printjson(indexes);
		  });
                    }
                );
            }
        }
    );

    if (isMongos()) {
        sh.status();
    } else {
        printjson(db.adminCommand('replSetGetStatus')); 
    }
    db.isMaster();
    print(getHeader("Logs",FILLER,LENGTH));
    db.adminCommand({'getLog': '*'})["names"].forEach(
        function (element, array, index) {
            db.adminCommand({'getLog': element})["log"].forEach(
                function (element, array, index) {
                    print(element);
                }
            );
        }
    );
}

printExtraDiagnosticsInfo();
