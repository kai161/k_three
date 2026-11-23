#!/usr/bin/env bash
set -euo pipefail

base_url="${BASE_URL:-http://localhost:8080}"
cookie_jar="$(mktemp)"
trap 'rm -f "$cookie_jar"' EXIT

request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  curl --fail-with-body --silent --show-error \
    --cookie "$cookie_jar" --cookie-jar "$cookie_jar" \
    --header 'Content-Type: application/json' \
    --request "$method" ${data:+--data "$data"} \
    "$base_url$path"
}

request GET /healthz
request POST /api/session

for day in 1 2 3 4 5 6 7; do
  request POST /api/replies "{\"text\":\"冒烟测试第 ${day} 天\"}" >/dev/null
  request POST /api/entries "{\"day\":${day},\"mood\":\"平静\",\"ritual\":\"揉掉\"}" >/dev/null
  if [ "$day" -eq 2 ] || [ "$day" -eq 7 ]; then
    request POST /api/feedback "{\"day\":${day},\"text\":\"公测冒烟反馈第 ${day} 天\"}" >/dev/null
  fi
done

review="$(request GET /api/review)"
REVIEW="$review" python3 -c 'import json, os; data=json.loads(os.environ["REVIEW"]); assert data["completedDays"] == 7; assert data["mostFrequentMood"] == "平静"; print("smoke: public beta seven-day flow and feedback passed")'
