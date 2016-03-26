
  var BASE_PORT=2800;
  var HOSTNAME="telecaster.local";
  rs.initiate();
  var prefix = HOSTNAME+":"+BASE_PORT;
  [ prefix+0 ,prefix+1, prefix+2, prefix+3].forEach(
      function (element, array, index) {
          if (element==HOSTNAME+":"+BASE_PORT+3) {
              rs.add(element,true);
          } else {
              rs.add(element);
          }
          rs.config();
      }
  ); 
