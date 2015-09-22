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
# get the header.
$TOOLNAME -n 1 $interval | grep insert > $dest.headers
# now get the capture without header lines.
$TOOLNAME --noheaders -n $duration $interval | gzip -c > $dest &
echo $! > $dest.pid
pid=$!
# save the pid so we can monitor disk space while the tool runs, and
# terminate it if needed. 
monitor_disk_space $pid

rm -f $dest.pid 2>/dev/null

   

