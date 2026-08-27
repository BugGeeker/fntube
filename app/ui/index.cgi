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
    # 保留原始查询参数
    if [ -n "${QUERY_STRING}" ]; then
      BACKEND_URL="${BACKEND_URL}?${QUERY_STRING}"
    fi

    METHOD="${REQUEST_METHOD:-GET}"
    HDR_FILE="/tmp/fntube_hdr_$$"
    BODY_FILE="/tmp/fntube_body_$$"

    # GET/HEAD 不读 body，避免 --data-binary @- 导致 curl 挂起
    if [ "$METHOD" = "GET" ] || [ "$METHOD" = "HEAD" ]; then
      curl -s -D "$HDR_FILE" -o "$BODY_FILE" "$BACKEND_URL" 2>/dev/null
    else
      curl -s -D "$HDR_FILE" -o "$BODY_FILE" -X "$METHOD" \
        ${CONTENT_TYPE:+-H "Content-Type: ${CONTENT_TYPE}"} \
        ${CONTENT_LENGTH:+-H "Content-Length: ${CONTENT_LENGTH}"} \
        --data-binary @- \
        "$BACKEND_URL" 2>/dev/null
    fi

    # 解析后端响应状态码
    STATUS_CODE=$(head -1 "$HDR_FILE" 2>/dev/null | grep -o '[0-9]\{3\}')
    if [ -n "$STATUS_CODE" ] && [ "$STATUS_CODE" != "200" ]; then
      echo "Status: ${STATUS_CODE}"
    fi

    # 输出响应头（跳过 HTTP 状态行），过滤 hop-by-hop 头。
    # curl 的头文件末尾已包含空行，不能再输出换行，否则会破坏图片二进制文件头。
    tail -n +2 "$HDR_FILE" 2>/dev/null | \
      grep -v -i '^Transfer-Encoding:' | \
      grep -v -i '^Connection:' | \
      grep -v -i '^Content-Length:'
    cat "$BODY_FILE" 2>/dev/null

    rm -f "$HDR_FILE" "$BODY_FILE" 2>/dev/null
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
  webp) mime="image/webp" ;;
  *) mime="application/octet-stream" ;;
esac

echo "Content-Type: $mime"
echo ""
cat "$TARGET_FILE"
