
_MST_MONGOD=${MST_MONGOD:-mongod}
_MST_MONGOS=${MST_MONGOS:-mongos}
_MST_MONGO=${MST_MONGO:-mongo}

export mst_DBPATH_ROOT=~/mongo-summary-tests/

export mst_BASE_PORT=28000

export mst_HOSTNAME="telecaster.local"

function mst_createDatadir()
{
   test -d $1 && rm -rf $mst_DBPATH_ROOT/$1
   mkdir -p $mst_DBPATH_ROOT/$1
}

function mst_startInstance()
{
    program=$1
    dbpath=$2
    port=$3
    dbpath_arg=""
    mst_createDatadir $dbpath
    [ $(echo $program|grep -c mongos) -eq 0 ] && dbpath_arg="--dbpath $mst_DBPATH_ROOT/$dbpath"
    shift; shift; shift
    $program $dbpath_arg --port=$port --logpath $mst_DBPATH_ROOT/$dbpath/log --fork --pidfilepath $mst_DBPATH_ROOT/$dbpath/pid $*
    sleep 5
}

function mst_stopInstance()
{
    kill $(cat $mst_DBPATH_ROOT/$1/pid)
    rm -rf $mst_DBPATH_ROOT/$1
}

function mst_validateOutput()
{
   f=$1; shift
   ok=0
   while [ -n "$1" ]; do
       grep "$1" $f>/dev/null || {
	   ok=1
	   echo "$1 not found on $f">&2
       }
      shift
   done
   return $ok
}

function mst_test_standalone_mongod()
{
    mst_startInstance $_MST_MONGOD standalone $mst_BASE_PORT
    sh mongo-summary.sh --extra --port $mst_BASE_PORT > test_standalone_mongod.result.txt
    mst_stopInstance standalone
    mst_validateOutput test_standalone_mongod.result.txt "Standalone mongod" version backgroundFlushing "Server Parameters" argv "local has" Logs
}

function mst_test_replica_set()
{
    nodes="primary secondary1 secondary2 arbiter"
    port_offset=0
    for node in $nodes; do
        mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset)) --replSet "test"
        port_offset=$((port_offset + 1))
    done

    $_MST_MONGO --port $mst_BASE_PORT mongo-summary-test-replset.js
    echo "Sleeping 2 seconds waiting for the replica set configuration to get applied" && sleep 2
    sh mongo-summary.sh --extra --port $mst_BASE_PORT > test_replica_set.result.txt
    for node in $nodes; do
        mst_stopInstance $node
    done
    mst_validateOutput test_replica_set.result.txt "Node is PRIMARY" 28002 connections "Server Parameters" argv "local has" 
}

function mst_test_shard_pair()
{
    nodes="shard1 shard2 config1 config2 config3 mongos"
    port_offset=0
    config1_port=$((mst_BASE_PORT + 2))
    config2_port=$((mst_BASE_PORT + 3))
    config3_port=$((mst_BASE_PORT + 4))
    mongos_port=$((mst_BASE_PORT + 5))
    for node in $nodes; do 
        if [ $(echo $node|grep -c config) -gt 0 ]; then
            mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset)) --configsvr
        elif [ "$node" == "mongos" ]; then
            mst_startInstance $_MST_MONGOS $node $((mst_BASE_PORT + port_offset)) --configdb "$mst_HOSTNAME:$config1_port,$mst_HOSTNAME:$config2_port,$mst_HOSTNAME:$config3_port"
        else
            mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset))
        fi
        port_offset=$((port_offset + 1))
    done

  for port in $mst_BASE_PORT $((mst_BASE_PORT + 1)); do
$_MST_MONGO --port $mongos_port --eval "sh.addShard(\"$mst_HOSTNAME:$port\")"
  done

$_MST_MONGO --port $mongos_port --eval "sh.enableSharding(\"test\")" 
$_MST_MONGO $mst_HOSTNAME:$mongos_port/test --eval 'db.test.insert({test:true})'

sh mongo-summary.sh --extra --port $mongos_port > test_sharded_cluster.result.txt
for node in $nodes; do
    mst_stopInstance $node
done
mst_validateOutput test_sharded_cluster.result.txt "Sharding Summary" "Detected 2 shards" "Shards detail" "operations in progress"

}

function mst_test_shard_replset()
{
    nodes="shard1_1 shard1_2 shard2_1 shard2_2 config1 config2 config3 mongos"
    port_offset=0
    config1_port=$((mst_BASE_PORT + 4))
    config2_port=$((mst_BASE_PORT + 5))
    config3_port=$((mst_BASE_PORT + 6))
    mongos_port=$((mst_BASE_PORT + 7))
    for node in $nodes; do 
        if [ $(echo $node|grep -c config) -gt 0 ]; then
            mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset)) --configsvr
        elif [ "$node" == "mongos" ]; then
            mst_startInstance $_MST_MONGOS $node $((mst_BASE_PORT + port_offset)) --configdb "$mst_HOSTNAME:$config1_port,$mst_HOSTNAME:$config2_port,$mst_HOSTNAME:$config3_port"
        elif [ $(echo $node|grep -c shard1) -gt 0 ]; then
            mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset)) --replSet rs1
        else
            mst_startInstance $_MST_MONGOD $node $((mst_BASE_PORT + port_offset)) --replSet rs2
        fi
        port_offset=$((port_offset + 1))
    done

    $_MST_MONGO --port $mst_BASE_PORT mongo-summary-test-sharded-rs1.js
    sleep 1
    $_MST_MONGO --port $((mst_BASE_PORT+2)) mongo-summary-test-sharded-rs2.js
    sleep 1
    $_MST_MONGO --port $mst_BASE_PORT mongo-summary-test-sharded-rs1.js
    sleep 1
    $_MST_MONGO --port $((mst_BASE_PORT+2)) mongo-summary-test-sharded-rs2.js
    sleep 1
    for port in $mst_BASE_PORT $((mst_BASE_PORT + 1)); do
        $_MST_MONGO --port $mongos_port --eval "sh.addShard(\"rs1/$mst_HOSTNAME:$port\")"
    done
    for port in $((mst_BASE_PORT + 2)) $((mst_BASE_PORT + 3)); do
        $_MST_MONGO --port $mongos_port --eval "sh.addShard(\"rs2/$mst_HOSTNAME:$port\")"
    done
    
    $_MST_MONGO --port $mongos_port --eval "sh.enableSharding(\"test\")"
    $_MST_MONGO$mst_HOSTNAME:$mongos_port/test --eval 'db.test.insert({test:true})'
    
    sh mongo-summary.sh --extra --port $mongos_port > test_sharded_cluster_replset.result.txt
    for node in $nodes; do
        mst_stopInstance $node
    done
}
