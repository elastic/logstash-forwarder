#!/bin/sh

echo "$@"
set -- `getopt o: "$@"`
echo "$@"

while [ $# -gt 0 ] ; do
  case "$1" in
    -o)
      OUTPUT=$2
      shift
      ;;
    --) 
      shift
      break
      ;;
  esac

  shift
done

URL="$1"
echo "URL: $URL"

if which wget > /dev/null 2>&1 ; then
  exec wget -O "$OUTPUT" "$URL"
elif which curl > /dev/null 2>&1 ; then
  exec curl -s -o "$OUTPUT" "$URL"
elif which GET > /dev/null 2>&1 ; then
  exec GET "$URL" > "$OUTPUT"
else
  echo "no http download tool found. cannot fetch."
  exit 1
fi

