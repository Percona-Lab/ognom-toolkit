msg()
{
    echo $*>&2
}

init_dbpath()
{
    [ $# -eq 1 ] || {
	cat <<EOF>&2
    usage: init_dbpath <directory>
    Initializes (creates if needed, removes data if any exists) a directory for use as an argument to --dbpath.
EOF
    }
    test -d $1 || mkdir -p $1
    rm -rf $1/*
    touch $1/mongod.pid
    chmod a+w $1/mongod.pid
}

start_mongod()
{
    [ -z "$MONGOD" ] && MONGOD=mongod
    [ $# -ge 2 ] || {
	cat <<EOF>&2
    usage: start_mongod <dbpath> <port> [replSet|configsvr]
    Starts a new mongod instance with the provided args for --dbpath and --port (and optionally indicates a replica set name, or starts the instance as a config server). 
    Initializes the datadir if needed. 
EOF
    }
    dbpath=$1; port=$2; replSet=$3
    [ -n "$replSet" ] && {
	[ "$replSet" == "configsvr" ] && replSet="--configsvr" || replSet="--replSet $replSet"
    }
    $MONGOD --rest --httpinterface --dbpath $dbpath --port $port --logpath $dbpath/mongod.log $replSet --pidfilepath $dbpath/mongod.pid --fork
}

start_mongos()
{
    [ -z "$MONGOS" ] && MONGOS=mongos
    [ $# -eq 3 ] || {
	cat <<EOF>&2
    usage: start_mongos <dpath> <port> <configdb>
    Starts a new mongos instance with the specified arguments. 
EOF
    }
    dbpath=$1; port=$2; configdb=$3
    # we intentionally use mongod.pid below so we can use stop_mongo
    $MONGOS --setParameter enableTestCommands=1 --configdb $configdb --logpath $dbpath/mongos.log --port $port --pidfilepath $dbpath/mongod.pid --fork
}

stop_mongo()
{
    [ $# -eq 1 ] || {
cat <<EOF>&2
    usage: stop_mongo <dbpath>
    Sends SIGTERM to the mongod/mongos instance running out of the specified datadir. 
EOF
    }
    kill $(cat $1/mongod.pid)
}
