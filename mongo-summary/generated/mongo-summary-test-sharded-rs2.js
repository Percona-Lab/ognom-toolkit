
    var BASE_PORT=2800;
    var HOSTNAME="telecaster";
    rs.initiate();
    var prefix = HOSTNAME+":"+BASE_PORT;
    [ prefix+2, prefix+3 ].forEach(
        function (element, array, index) {
            rs.add(element);
            rs.config();
        }
    ); 
