#!/bin/bash

BASE_PATH="/var/apps/FnTube/target/www"
URI_NO_QUERY="${REQUEST_URI%%\?*}"
REL_PATH="/"

case "$URI_NO_QUERY" in
  *index.cgi*)
    REL_PATH="${URI_NO_QUERY#*index.cgi}"
    ;;
esac

if [ -z "$REL_PATH" ] || [ "$REL_PATH" = "/" ]; then
  REL_PATH="/index.html"
fi

# API 请求转发到后端服务
case "$REL_PATH" in
  /api/*)
    PORT="${FN_APP_PORT:-12786}"
    BACKEND_URL="http://127.0.0.1:${PORT}${REL_PATH}"
    # 传递请求方法和 body，输出响应头和响应体
    curl -s -i -X "${REQUEST_METHOD:-GET}" \
      ${CONTENT_TYPE:+-H "Content-Type: ${CONTENT_TYPE}"} \
      ${CONTENT_LENGTH:+-H "Content-Length: ${CONTENT_LENGTH}"} \
      --data-binary @- \
      "${BACKEND_URL}" 2>/dev/null
    exit 0
    ;;
esac

TARGET_FILE="${BASE_PATH}${REL_PATH}"

if echo "$TARGET_FILE" | grep -q '\.\.'; then
  echo "Status: 400 Bad Request"
  echo "Content-Type: text/plain; charset=utf-8"
  echo ""
  echo "Bad Request"
  exit 0
fi

if [ ! -f "$TARGET_FILE" ]; then
  echo "Status: 404 Not Found"
  echo "Content-Type: text/plain; charset=utf-8"
  echo ""
  echo "404 Not Found"
  exit 0
fi

case "${TARGET_FILE##*.}" in
  html|htm) mime="text/html; charset=utf-8" ;;
  css) mime="text/css; charset=utf-8" ;;
  js) mime="application/javascript; charset=utf-8" ;;
  png) mime="image/png" ;;
  jpg|jpeg) mime="image/jpeg" ;;
  svg) mime="image/svg+xml" ;;
  *) mime="application/octet-stream" ;;
esac

echo "Content-Type: $mime"
echo ""
cat "$TARGET_FILE"
