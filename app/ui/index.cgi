#!/bin/bash

URI_NO_QUERY="${REQUEST_URI%%\?*}"
REL_PATH="/"

case "$URI_NO_QUERY" in
  *index.cgi*)
    REL_PATH="${URI_NO_QUERY#*index.cgi}"
    ;;
esac

if [ -z "$REL_PATH" ]; then
  REL_PATH="/"
fi

LOCATION="/app/FnTube${REL_PATH}"
if [ -n "${QUERY_STRING}" ]; then
  LOCATION="${LOCATION}?${QUERY_STRING}"
fi

echo "Status: 302 Found"
echo "Location: ${LOCATION}"
echo "Cache-Control: no-store"
echo ""
