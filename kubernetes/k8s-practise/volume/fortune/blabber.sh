#!/usr/bin/env sh
trap "exit" SIGINT SIGTERM

OUTPUT_FILE=${OUTPUT_FILE:-/var/www/index.html}
mkdir -p "$(dirname "${OUTPUT_FILE}")"

while :; do
  echo "[$(date)] Writing fortune to ${OUTPUT_FILE} ..."
  fortune >> "${OUTPUT_FILE}"
  sleep 10
done
