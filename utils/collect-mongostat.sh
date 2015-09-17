#!/bin/bash

# This will be used to stop the script if more than MAX_USED_DISK_PCT percentage of the drive where data is being
# collected is in use.  
# It can be overriden from the environment so there is no need to modify the script 
[ -z "$MAX_USED_DISK_PCT" ] && MAX_USED_DISK_PCT=90

# Forced tested by creating a large file with dd while the script is running: 
# telecaster:utils fernandoipar$ env MAX_USED_DISK_PCT=85 ./collect-mongostat.sh 1 60
# Terminating due to used disk space (threshold is 85 %)

usage()
{
cat <<EOF>&2
   usage: $0 <interval> <duration>
   Run mongostat <interval> for <duration> seconds (forever if <duration> is 0)
   Output is saved on a timestamped file in the current directory
   Arguments are *not* validated so it's up to you to pass only integer values. 
EOF
}

ts()
{
    date "+%Y%m%d_%H%M%S"
}

[ $# -ne 2 ] && {
    usage
    exit 1
}

interval=$1
duration=$2

dest=mongostat_$(ts).gz
trap "rm -f $dest.pid" SIGINT SIGTERM SIGHUP
# get the header.
mongostat -n 1 $interval | grep insert |gzip -c > $dest.headers
# now get the capture without header lines.
mongostat --noheaders -n $duration $interval | gzip -c > $dest &
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

   

