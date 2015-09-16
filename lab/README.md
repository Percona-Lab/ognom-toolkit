These scripts are meant to test different mongodb topologies and should not be used in production.
You can also use the functions from common.sh to start your own custom topologies. 

By default, it will look for mongod/mongos on the path, but if you have multiple versions installed and would like to use a specific one, you can do this by exporting the MONGOD and MONGOS variables, setting them to the full path of the mongod and mongos you want to use, as in the next example:

    export MONGOD=/Users/fernandoipar/mongodb/mongodb-osx-x86_64-2.6.11/bin/mongod 
    ./start_single
