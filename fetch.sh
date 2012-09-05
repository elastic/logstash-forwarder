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

# If you need to pass any specific flags to any specific fetcher, set
# WGET_FLAGS, CURL_FLAGS, or GET_FLAGS in your environment accordingly.
if which wget > /dev/null 2>&1 ; then
  # Check if wget is a shitty version
  if ! wget -O /dev/null -q https://github.com/ ; then
    WGET_FLAGS="${WGET_FLAGS} --no-check-certificate"
  fi
  exec wget $WGET_FLAGS -O "$OUTPUT" "$URL"
elif which curl > /dev/null 2>&1 ; then
  exec curl $CURL_FLAGS -s -o "$OUTPUT" "$URL"
elif which GET > /dev/null 2>&1 ; then
  exec GET $GET_FLAGS "$URL" > "$OUTPUT"
else
  echo "no http download tool found. cannot fetch."
  exit 1
fi

