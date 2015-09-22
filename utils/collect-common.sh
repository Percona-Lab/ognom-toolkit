#!/bin/bash

# This will be used to stop the script if more than MAX_USED_DISK_PCT percentage of the drive where data is being
# collected is in use.  
# It can be overriden from the environment so there is no need to modify the script 
[ -z "$MAX_USED_DISK_PCT" ] && MAX_USED_DISK_PCT=90

# Forced tested on collect-mongostat by creating a large file with dd while the script is running: 
# telecaster:utils fernandoipar$ env MAX_USED_DISK_PCT=85 ./collect-mongostat.sh 1 60
# Terminating due to used disk space (threshold is 85 %)

TOOLNAME=$(basename $0|awk -F'-' '{print $2}'|sed 's/\.sh//g')

usage()
{
cat <<EOF>&2
   usage: $0 <interval> <duration>
   Run $TOOLNAME <interval> for <duration> seconds (forever if <duration> is 0)
   Output is saved on a timestamped file in the current directory
   Arguments are *not* validated so it's up to you to pass only integer values. 
EOF
}

ts()
{
    date "+%Y%m%d_%H%M%S"
}

monitor_disk_space()
{
pid=$1
[ -z "$pid" ] && echo "monitor_disk_space \$pid" && exit 1
while kill -SIGCONT $pid 2>/dev/null; do
    [ $(df $PWD|tail -1|awk '{print $5}'|tr -d '%') -ge $MAX_USED_DISK_PCT ] && {
        echo "Terminating due to used disk space (threshold is $MAX_USED_DISK_PCT %)"
        kill -SIGTERM $pid
        sleep 2
        kill -SIGKILL $pid 2>/dev/null
    }
done
}

dest=$TOOLNAME_$(ts).gz
trap "rm -f $dest.pid" SIGINT SIGTERM SIGHUP
