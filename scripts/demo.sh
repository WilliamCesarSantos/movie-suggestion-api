#!/bin/bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_EMAIL="william_cesar_santos@hotmail.com"
ADMIN_PASSWORD="123456"

sep() { echo; echo "══════════════════════════════════════════"; echo "  $1"; echo "══════════════════════════════════════════"; }

sep "1. Waiting for API..."
for i in $(seq 1 30); do
  if curl -sf "${BASE_URL}/api/v1/health" > /dev/null 2>&1; then
    echo "API is up!"
    break
  fi
  echo "Attempt $i/30..."
  sleep 3
done

sep "2. Login as admin"
LOGIN_RESP=$(curl -sf -X POST "${BASE_URL}/api/v1/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}")
TOKEN_ADMIN=$(echo "$LOGIN_RESP" | jq -r '.token')
echo "Admin token: ${TOKEN_ADMIN:0:50}..."

sep "3. Create user Alice"
ALICE_RESP=$(curl -sf -X POST "${BASE_URL}/api/v1/users" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"s3cr3t","roles":["users:read","users:write","suggestions:read","movies:read","movie-watch:write"]}')
ALICE_ID=$(echo "$ALICE_RESP" | jq -r '.id')
echo "Alice ID: ${ALICE_ID}"

sep "4. Login as Alice"
ALICE_LOGIN=$(curl -sf -X POST "${BASE_URL}/api/v1/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"s3cr3t"}')
TOKEN_ALICE=$(echo "$ALICE_LOGIN" | jq -r '.token')
echo "Alice token: ${TOKEN_ALICE:0:50}..."

sep "5. Trigger movie import"
curl -sf -X POST "${BASE_URL}/api/v1/movie-import" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" \
  -H "Content-Type: application/json" \
  -d '{"searchTerms":["inception","matrix"],"maxPages":1}'
echo "Import triggered"

sep "6. Waiting 15s for SQS consumer..."
sleep 15

sep "7. Get Alice (as Alice)"
curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}" \
  -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

sep "8. Get Alice (as Admin)"
curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" | jq .

sep "9. Finding a movie..."
MOVIE_SUGGESTIONS=$(curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}/suggestions" \
  -H "Authorization: Bearer ${TOKEN_ALICE}")
MOVIE_ID=$(echo "$MOVIE_SUGGESTIONS" | jq -r '.[0].id // empty')

if [ -z "$MOVIE_ID" ]; then
  echo "No movies found yet, skipping watch steps"
else
  sep "10. Record watched movie (liked)"
  curl -sf -X POST "${BASE_URL}/api/v1/movie/${MOVIE_ID}/watched" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" \
    -H "Content-Type: application/json" \
    -d '{"rating":8.5,"reaction":"liked"}' | jq .

  sep "11. Get suggestions for Alice"
  curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}/suggestions" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

  sep "12. Get suggestions (SERENDIPITY)"
  curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}/suggestions?algorithm=SERENDIPITY" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

  sep "13. Get movie details"
  curl -sf "${BASE_URL}/api/v1/movies/${MOVIE_ID}" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .
fi

sep "14. Invalid token (expect 401)"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/v1/movie/some-id/watched" \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"rating":5.0}')
echo "Status: ${STATUS}"
[ "$STATUS" = "401" ] && echo "✓ Got 401 as expected" || echo "✗ Expected 401, got ${STATUS}"

sep "15. Create user without auth (expect 401)"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com","password":"pass","roles":["users:read"]}')
echo "Status: ${STATUS}"
[ "$STATUS" = "401" ] && echo "✓ Got 401 as expected" || echo "✗ Expected 401, got ${STATUS}"

sep "16. Test 403: Create user with token lacking users:write"
LIMITED_RESP=$(curl -sf -X POST "${BASE_URL}/api/v1/users" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com","password":"pass123","roles":["movies:read"]}')
TOKEN_BOB_LOGIN=$(curl -sf -X POST "${BASE_URL}/api/v1/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"bob@example.com","password":"pass123"}')
TOKEN_BOB=$(echo "$TOKEN_BOB_LOGIN" | jq -r '.token')
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/v1/users" \
  -H "Authorization: Bearer ${TOKEN_BOB}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Charlie","email":"charlie@example.com","password":"pass","roles":["movies:read"]}')
echo "Status: ${STATUS}"
[ "$STATUS" = "403" ] && echo "✓ Got 403 as expected" || echo "✗ Expected 403, got ${STATUS}"

sep "17. Final health check"
curl -sf "${BASE_URL}/api/v1/health" | jq .

echo
echo "Demo complete!"
