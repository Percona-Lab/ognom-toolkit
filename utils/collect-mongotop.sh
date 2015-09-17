#!/bin/bash

. $(dirname $0)/collect-common.sh

[ $# -ne 2 ] && {
    usage
    exit 1
}

interval=$1
duration=$2

dest=${TOOLNAME}_$(ts).gz
trap "rm -f $dest.pid" SIGINT SIGTERM SIGHUP
$TOOLNAME -n $duration $interval | gzip -c > $dest &
echo $! > $dest.pid
pid=$!
# save the pid so we can monitor disk space while the tool runs, and
# terminate it if needed. 
while kill -SIGCONT $pid 2>/dev/null; do
    [ $(df $PWD|tail -1|awk '{print $5}'|tr -d '%') -ge $MAX_USED_DISK_PCT ] && {
	echo "Terminating due to used disk space (threshold is $MAX_USED_DISK_PCT %)"
	kill -SIGTERM $pid
	sleep 2
	kill -SIGKILL $pid 2>/dev/null
    }
done

rm -f $dest.pid 2>/dev/null

   

