#!/bin/sh

set -- `getopt o: "$@"`

while [ $# -gt 0 ] ; do
  case "$1" in
    -o)
      OUTPUT=$2
      shift
      ;;
  esac
  shift
done

URL="$1"

if which wget > /dev/null 2>&1 ; then
  exec wget -q -O "$OUTPUT" "$URL"
elif which curl > /dev/null 2>&1 ; then
  exec curl -s -o "$OUTPUT" "$URL"
else
  echo "no http download tool found. cannot fetch."
  exit 1
fi

