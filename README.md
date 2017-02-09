# ognom-toolkit
ognom-toolkit is a set of utilities, functions and tests with the goal
of making the life of MongoDB/TokuMX administrators easier.

# name
ognom stands for mongo spelled backwards, with the following
advantages:
- it does not infringe any trademarks
- it shares the name with
http://www.worldofspectrum.org/infoseekid.cgi?id=0016326
- it sounds like it could have been part of the title of a Barret-era
Pink Floyd song

# contents

"lab" contains bash scripts and functions to help you quickly
start/stop test MongoDB topologies.

"load_generators" contains basic load generators to run against mongod

"mongo-summary" is a pt-mysql-summary inspired tool for MongoDB

"slowlog-generators" contains utilities to generate fake MySQL slow
query logs with MongoDB traffic, that can then be used with
pt-query-digest for workload analysis

"utils" contains small scripts that (for now) don't fall on any
specific category. 
