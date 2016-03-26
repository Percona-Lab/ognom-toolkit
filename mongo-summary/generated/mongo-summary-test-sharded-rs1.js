
  var BASE_PORT=2800;
  var HOSTNAME="telecaster.local";
  rs.initiate();
  var prefix = HOSTNAME+":"+BASE_PORT;
  [ prefix+0 ,prefix+1 ].forEach(
      function (element, array, index) {
          rs.add(element);
          rs.config();
      }
  ); 
