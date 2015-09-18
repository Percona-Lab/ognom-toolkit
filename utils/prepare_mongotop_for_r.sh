#!/bin/bash

[ $# -eq 0 ] && {
    cat <<EOF>&2
   This script expects to find a 'mongotop' file, created from a compressed collect-mongotop.sh capture. 
   Once you have done this (i.e. "gunzip -c <capture.gz> > mongotop"), run this again including any argument
   in order to bypass this check. 
EOF
exit 1
}


namespaces=$(awk '/ns    total/{matches++} matches==2{print; exit} matches>=1' mongotop|egrep -v 'ns    total|^$'|awk '{print $1}'|tr '\n' ' ')

for namespace in $namespaces; do
    grep $namespace mongotop > mongotop_$namespace
done
