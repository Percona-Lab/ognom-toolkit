
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
