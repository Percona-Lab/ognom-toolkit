
 extra=0
 [ "$1" == "--extra" ] && {
     extra=1
     shift
 }
 mongo mongo-summary.js $*
 [ $extra -eq 1 ] && mongo mongo-summary-extra.js $*
