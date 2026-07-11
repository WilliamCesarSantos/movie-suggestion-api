#!/bin/bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-william_cesar_santos@hotmail.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-123456}"
DEMO_TIMEOUT_SECONDS="${DEMO_TIMEOUT_SECONDS:-120}"
DEMO_POLL_INTERVAL_SECONDS="${DEMO_POLL_INTERVAL_SECONDS:-3}"
DEMO_VERBOSE="${DEMO_VERBOSE:-0}"
DEMO_EMAIL_SUFFIX="${DEMO_EMAIL_SUFFIX:-}"
ALICE_EMAIL="${ALICE_EMAIL:-alice${DEMO_EMAIL_SUFFIX}@example.com}"
BOB_EMAIL="${BOB_EMAIL:-bob${DEMO_EMAIL_SUFFIX}@example.com}"

sep() { echo; echo "══════════════════════════════════════════"; echo "  $1"; echo "══════════════════════════════════════════"; }
log_info() { echo "[INFO] $1"; }
log_ok() { echo "[OK]   $1"; }
log_warn() { echo "[WARN] $1"; }
log_err() { echo "[ERR]  $1"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    log_err "Missing dependency: $1"
    exit 1
  }
}

new_correlation_id() {
  printf "%s-%s" "$(date +%s)" "$RANDOM"
}

HTTP_STATUS=""
HTTP_BODY=""
HTTP_CID=""
request_json() {
  local method="$1"
  local url="$2"
  local token="${3:-}"
  local body="${4:-}"
  local cid="${5:-$(new_correlation_id)}"

  local auth_args=()
  if [ -n "$token" ]; then
    auth_args=(-H "Authorization: Bearer ${token}")
  fi

  local tmp_file
  tmp_file="$(mktemp)"
  local status

  if [ -n "$body" ]; then
    status=$(curl -sS -o "$tmp_file" -w "%{http_code}" -X "$method" "$url" \
      -H "Content-Type: application/json" \
      -H "X-Correlation-ID: ${cid}" \
      "${auth_args[@]}" \
      -d "$body")
  else
    status=$(curl -sS -o "$tmp_file" -w "%{http_code}" -X "$method" "$url" \
      -H "X-Correlation-ID: ${cid}" \
      "${auth_args[@]}")
  fi

  HTTP_STATUS="$status"
  HTTP_BODY="$(cat "$tmp_file")"
  HTTP_CID="$cid"
  rm -f "$tmp_file"

  if [ "$DEMO_VERBOSE" = "1" ]; then
    log_info "CID=${HTTP_CID} METHOD=${method} URL=${url} STATUS=${HTTP_STATUS}"
  fi
}

CHECKS_TOTAL=0
CHECKS_FAILED=0
assert_status() {
  CHECKS_TOTAL=$((CHECKS_TOTAL + 1))
  local got="$1"
  local expected="$2"
  local label="$3"
  if [ "$got" = "$expected" ]; then
    log_ok "${label} (status=${got})"
  else
    log_err "${label} (expected=${expected} got=${got})"
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
  fi
}

require_cmd curl
require_cmd jq

sep "1. Waiting for API..."
for i in $(seq 1 30); do
  request_json "GET" "${BASE_URL}/api/v1/health"
  if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "503" ]; then
    log_ok "API is up (status=${HTTP_STATUS}, correlationId=${HTTP_CID})"
    break
  fi
  log_info "Attempt $i/30..."
  sleep 3
done

sep "2. Login as admin"
request_json "POST" "${BASE_URL}/api/v1/login" "" "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}"
if [ "$HTTP_STATUS" != "200" ]; then
  log_err "Admin login failed (status=${HTTP_STATUS}, cid=${HTTP_CID})"
  echo "$HTTP_BODY"
  exit 1
fi
TOKEN_ADMIN=$(echo "$HTTP_BODY" | jq -r '.token')
log_ok "Admin token acquired: ${TOKEN_ADMIN:0:50}..."

sep "3. Ensure user Alice exists"
request_json "POST" "${BASE_URL}/api/v1/users" "${TOKEN_ADMIN}" "{\"name\":\"Alice\",\"email\":\"${ALICE_EMAIL}\",\"password\":\"s3cr3t\",\"roles\":[\"users:read\",\"movies:read\",\"movies-watch:write\"]}"
if [ "$HTTP_STATUS" = "201" ]; then
  ALICE_ID=$(echo "$HTTP_BODY" | jq -r '.id')
  log_ok "Alice created: ${ALICE_ID}"
elif [ "$HTTP_STATUS" = "409" ]; then
  log_warn "Alice already exists, resolving id by email"
  request_json "GET" "${BASE_URL}/api/v1/users?email=${ALICE_EMAIL}&page=1&pageSize=1" "${TOKEN_ADMIN}"
  if [ "$HTTP_STATUS" != "200" ]; then
    log_err "Failed to resolve Alice id (status=${HTTP_STATUS})"
    echo "$HTTP_BODY"
    exit 1
  fi
  ALICE_ID=$(echo "$HTTP_BODY" | jq -r '.data[0].id // empty')
  if [ -z "$ALICE_ID" ]; then
    log_err "Alice id not found in listUsers response"
    exit 1
  fi
  log_ok "Alice resolved: ${ALICE_ID}"
else
  log_err "Failed to ensure Alice (status=${HTTP_STATUS}, cid=${HTTP_CID})"
  echo "$HTTP_BODY"
  exit 1
fi

sep "4. Login as Alice"
request_json "POST" "${BASE_URL}/api/v1/login" "" "{\"email\":\"${ALICE_EMAIL}\",\"password\":\"s3cr3t\"}"
if [ "$HTTP_STATUS" != "200" ]; then
  log_err "Alice login failed (status=${HTTP_STATUS}, cid=${HTTP_CID})"
  echo "$HTTP_BODY"
  exit 1
fi
TOKEN_ALICE=$(echo "$HTTP_BODY" | jq -r '.token')
log_ok "Alice token acquired: ${TOKEN_ALICE:0:50}..."

sep "5. Trigger movie import"
request_json "POST" "${BASE_URL}/api/v1/movies-import" "${TOKEN_ADMIN}" '{"searchTerms":["inception", "matrix", "interstellar", "tropa", "compadecida", "chefão", "pulp fiction"],"maxPages":5}'
if [ "$HTTP_STATUS" != "202" ]; then
  log_err "Import trigger failed (status=${HTTP_STATUS}, cid=${HTTP_CID})"
  echo "$HTTP_BODY"
  exit 1
fi
log_ok "Import triggered"

sep "6. Waiting for imported movies"
start_ts=$(date +%s)
ready=0
while true; do
  request_json "GET" "${BASE_URL}/api/v1/movies?limit=1" "${TOKEN_ALICE}"
  if [ "$HTTP_STATUS" = "200" ]; then
    count=$(echo "$HTTP_BODY" | jq '.data | length')
    if [ "$count" -gt 0 ]; then
      ready=1
      break
    fi
  fi
  now_ts=$(date +%s)
  elapsed=$((now_ts - start_ts))
  if [ "$elapsed" -ge "$DEMO_TIMEOUT_SECONDS" ]; then
    break
  fi
  sleep "$DEMO_POLL_INTERVAL_SECONDS"
done

if [ "$ready" -eq 1 ]; then
  log_ok "Movies available"
else
  log_warn "No movies available after ${DEMO_TIMEOUT_SECONDS}s; continuing demo"
fi

sep "7. List movies and record watched for Alice"
request_json "GET" "${BASE_URL}/api/v1/movies?limit=20" "${TOKEN_ALICE}"
if [ "$HTTP_STATUS" != "200" ]; then
  log_err "Failed to list movies (status=${HTTP_STATUS}, cid=${HTTP_CID})"
  echo "$HTTP_BODY"
  exit 1
fi
MOVIE_LIST="$HTTP_BODY"
MOVIE_IDS=($(echo "$MOVIE_LIST" | jq -r '.data[].id' | head -10))

REACTIONS=("liked" "liked" "liked" "liked" "liked" "liked" "disliked" "neutral" "liked" "disliked")
RATINGS=("8.5" "9.0" "7.5" "8.0" "6.5" "9.5" "4.0" "6.0" "8.8" "3.5")

if [ ${#MOVIE_IDS[@]} -eq 0 ]; then
  log_warn "No movies imported yet, skipping watch registrations"
else
  log_info "Registering ${#MOVIE_IDS[@]} watched movies for Alice..."
  for i in "${!MOVIE_IDS[@]}"; do
    MID="${MOVIE_IDS[$i]}"
    REACTION="${REACTIONS[$i]}"
    RATING="${RATINGS[$i]}"
    request_json "POST" "${BASE_URL}/api/v1/movies/${MID}/watched" "${TOKEN_ALICE}" "{\"rating\":${RATING},\"reaction\":\"${REACTION}\"}"
    if [ "$HTTP_STATUS" = "200" ]; then
      log_ok "Movie ${MID} -> ${REACTION} (${RATING})"
    else
      log_warn "Failed to register watched for ${MID} (status=${HTTP_STATUS})"
    fi
  done
fi

sep "8. Get Alice (as Alice)"
request_json "GET" "${BASE_URL}/api/v1/users/${ALICE_ID}" "${TOKEN_ALICE}"
if [ "$HTTP_STATUS" = "200" ]; then
  echo "$HTTP_BODY" | jq .
else
  log_warn "Get Alice as Alice failed (status=${HTTP_STATUS})"
fi

sep "9. Get Alice (as Admin)"
request_json "GET" "${BASE_URL}/api/v1/users/${ALICE_ID}" "${TOKEN_ADMIN}"
if [ "$HTTP_STATUS" = "200" ]; then
  echo "$HTTP_BODY" | jq .
else
  log_warn "Get Alice as Admin failed (status=${HTTP_STATUS})"
fi

sep "10. Get recommended movies for Alice (auto-selected algorithm)"
request_json "GET" "${BASE_URL}/api/v1/movies" "${TOKEN_ALICE}"
if [ "$HTTP_STATUS" = "200" ]; then
  echo "$HTTP_BODY" | jq .
else
  log_warn "Get recommendations failed (status=${HTTP_STATUS})"
fi

sep "11. Finding a movie for further steps..."
request_json "GET" "${BASE_URL}/api/v1/movies" "${TOKEN_ALICE}"
if [ "$HTTP_STATUS" != "200" ]; then
  log_warn "Cannot fetch recommendations for follow-up steps (status=${HTTP_STATUS})"
  MOVIE_RECOMMENDATIONS='{"data":[]}'
else
  MOVIE_RECOMMENDATIONS="$HTTP_BODY"
fi
MOVIE_ID=$(echo "$MOVIE_RECOMMENDATIONS" | jq -r '.data[0].id // empty')

if [ -z "$MOVIE_ID" ]; then
  log_warn "No movies found yet, skipping watch steps"
else
  sep "12. Record one more watched movie (liked)"
  request_json "POST" "${BASE_URL}/api/v1/movies/${MOVIE_ID}/watched" "${TOKEN_ALICE}" '{"rating":8.5,"reaction":"liked"}'
  if [ "$HTTP_STATUS" = "200" ]; then
    echo "$HTTP_BODY" | jq .
  else
    log_warn "Record watched failed (status=${HTTP_STATUS})"
  fi

  sep "13. Get recommended movies for Alice"
  request_json "GET" "${BASE_URL}/api/v1/movies" "${TOKEN_ALICE}"
  if [ "$HTTP_STATUS" = "200" ]; then
    echo "$HTTP_BODY" | jq .
  else
    log_warn "Recommendations after watched failed (status=${HTTP_STATUS})"
  fi

  sep "14. Get recommended movies (SERENDIPITY)"
  request_json "GET" "${BASE_URL}/api/v1/movies?algorithm=SERENDIPITY" "${TOKEN_ALICE}"
  if [ "$HTTP_STATUS" = "200" ]; then
    echo "$HTTP_BODY" | jq .
  else
    log_warn "Recommendations with SERENDIPITY failed (status=${HTTP_STATUS})"
  fi

  sep "15. Get movie details"
  request_json "GET" "${BASE_URL}/api/v1/movies/${MOVIE_ID}" "${TOKEN_ALICE}"
  if [ "$HTTP_STATUS" = "200" ]; then
    echo "$HTTP_BODY" | jq .
  else
    log_warn "Get movie details failed (status=${HTTP_STATUS})"
  fi
fi

sep "16. Invalid token (expect 401)"
request_json "POST" "${BASE_URL}/api/v1/movies/some-id/watched" "invalid-token" '{"rating":5.0}'
assert_status "$HTTP_STATUS" "401" "Invalid token must return 401"

sep "17. Create user without auth (expect 401)"
request_json "POST" "${BASE_URL}/api/v1/users" "" "{\"name\":\"Bob\",\"email\":\"${BOB_EMAIL}\",\"password\":\"pass\",\"roles\":[\"users:read\"]}"
assert_status "$HTTP_STATUS" "401" "Create user without auth must return 401"

sep "18. Test 403: Create user with token lacking users:write"
request_json "POST" "${BASE_URL}/api/v1/users" "${TOKEN_ADMIN}" "{\"name\":\"Bob\",\"email\":\"${BOB_EMAIL}\",\"password\":\"pass123\",\"roles\":[\"movies:read\"]}"
if [ "$HTTP_STATUS" != "201" ] && [ "$HTTP_STATUS" != "409" ]; then
  log_err "Failed to ensure Bob for 403 test (status=${HTTP_STATUS})"
  echo "$HTTP_BODY"
  exit 1
fi

request_json "POST" "${BASE_URL}/api/v1/login" "" "{\"email\":\"${BOB_EMAIL}\",\"password\":\"pass123\"}"
if [ "$HTTP_STATUS" != "200" ]; then
  log_err "Bob login failed for 403 test (status=${HTTP_STATUS})"
  echo "$HTTP_BODY"
  exit 1
fi
TOKEN_BOB=$(echo "$HTTP_BODY" | jq -r '.token')

request_json "POST" "${BASE_URL}/api/v1/users" "${TOKEN_BOB}" '{"name":"Charlie","email":"charlie@example.com","password":"pass","roles":["movies:read"]}'
assert_status "$HTTP_STATUS" "403" "User without users:write must receive 403 when creating user"

sep "19. Final health check"
request_json "GET" "${BASE_URL}/api/v1/health"
if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "503" ]; then
  echo "$HTTP_BODY" | jq .
else
  log_warn "Final health check returned status ${HTTP_STATUS}"
fi

echo
echo "Demo complete!"
echo "Checks total: ${CHECKS_TOTAL}"
echo "Checks failed: ${CHECKS_FAILED}"

if [ "$CHECKS_FAILED" -gt 0 ]; then
  exit 1
fi
