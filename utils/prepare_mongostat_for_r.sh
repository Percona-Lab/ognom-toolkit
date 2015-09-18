#!/bin/bash

[ $# -eq 0 ] && {
    cat <<EOF>&2
   This script expects to find a 'mongostat' file, created from a compressed collect-mongostat.sh capture. 
   Once you have done this (i.e. "gunzip -c <capture.gz> > mongostat"), run this again including any argument
   in order to bypass this check. 
EOF
exit 1
}

cleanup_data()
{
    while read line; do 
	for word in $line; do 
	    echo $word | grep '|'>/dev/null && echo -n "$word " && continue
	    echo $word | grep ':'>/dev/null && echo -n "$word " && continue
	    echo -n "$(echo $word|bc) "
	done
	echo
    done 
}

sed 's/  */ /g' < mongostat | \
    tr -d '*' | \
    sed 's/M/*1024*1024/g' | \
    sed 's/k/*1024/g' | \
    sed 's/G/*1024*1024*1024/g' | \
    tr -d 'b' | cleanup_data > mongostat_curated 
