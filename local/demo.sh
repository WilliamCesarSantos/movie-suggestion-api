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
  -d '{"name":"Alice","email":"alice@example.com","password":"s3cr3t","roles":["users:read","movies:read","movies-watch:write"]}')
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
  -d '{"searchTerms":["inception", "matrix", "interstellar", "tropa", "compadecida", "chefão", "pulp fiction"],"maxPages":5}'
echo "Import triggered"

sep "6. Waiting 15s for SQS consumer..."
sleep 15

sep "7. List movies and record watched for Alice"
MOVIE_LIST=$(curl -sf "${BASE_URL}/api/v1/movies?limit=20" \
  -H "Authorization: Bearer ${TOKEN_ALICE}")
MOVIE_IDS=($(echo "$MOVIE_LIST" | jq -r '.data[].id' | head -10))

REACTIONS=("liked" "liked" "liked" "liked" "liked" "liked" "disliked" "neutral" "liked" "disliked")
RATINGS=("8.5" "9.0" "7.5" "8.0" "6.5" "9.5" "4.0" "6.0" "8.8" "3.5")

if [ ${#MOVIE_IDS[@]} -eq 0 ]; then
  echo "No movies imported yet, skipping watch registrations"
else
  echo "Registering ${#MOVIE_IDS[@]} watched movies for Alice..."
  for i in "${!MOVIE_IDS[@]}"; do
    MID="${MOVIE_IDS[$i]}"
    REACTION="${REACTIONS[$i]}"
    RATING="${RATINGS[$i]}"
    curl -sf -X POST "${BASE_URL}/api/v1/movies/${MID}/watched" \
      -H "Authorization: Bearer ${TOKEN_ALICE}" \
      -H "Content-Type: application/json" \
      -d "{\"rating\":${RATING},\"reaction\":\"${REACTION}\"}" > /dev/null
    echo "  ✓ Movie ${MID} → ${REACTION} (${RATING})"
  done
fi

sep "8. Get Alice (as Alice)"
curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}" \
  -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

sep "9. Get Alice (as Admin)"
curl -sf "${BASE_URL}/api/v1/users/${ALICE_ID}" \
  -H "Authorization: Bearer ${TOKEN_ADMIN}" | jq .

sep "10. Get recommended movies for Alice (auto-selected algorithm)"
curl -sf "${BASE_URL}/api/v1/movies" \
  -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

sep "11. Finding a movie for further steps..."
MOVIE_RECOMMENDATIONS=$(curl -sf "${BASE_URL}/api/v1/movies" \
  -H "Authorization: Bearer ${TOKEN_ALICE}")
MOVIE_ID=$(echo "$MOVIE_RECOMMENDATIONS" | jq -r '.data[0].id // empty')

if [ -z "$MOVIE_ID" ]; then
  echo "No movies found yet, skipping watch steps"
else
  sep "12. Record one more watched movie (liked)"
  curl -sf -X POST "${BASE_URL}/api/v1/movies/${MOVIE_ID}/watched" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" \
    -H "Content-Type: application/json" \
    -d '{"rating":8.5,"reaction":"liked"}' | jq .

  sep "13. Get recommended movies for Alice"
  curl -sf "${BASE_URL}/api/v1/movies" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

  sep "14. Get recommended movies (SERENDIPITY)"
  curl -sf "${BASE_URL}/api/v1/movies?algorithm=SERENDIPITY" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .

  sep "15. Get movie details"
  curl -sf "${BASE_URL}/api/v1/movies/${MOVIE_ID}" \
    -H "Authorization: Bearer ${TOKEN_ALICE}" | jq .
fi

sep "16. Invalid token (expect 401)"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/v1/movies/some-id/watched" \
  -H "Authorization: Bearer invalid-token" \
  -H "Content-Type: application/json" \
  -d '{"rating":5.0}')
echo "Status: ${STATUS}"
[ "$STATUS" = "401" ] && echo "✓ Got 401 as expected" || echo "✗ Expected 401, got ${STATUS}"

sep "17. Create user without auth (expect 401)"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com","password":"pass","roles":["users:read"]}')
echo "Status: ${STATUS}"
[ "$STATUS" = "401" ] && echo "✓ Got 401 as expected" || echo "✗ Expected 401, got ${STATUS}"

sep "18. Test 403: Create user with token lacking users:write"
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

sep "19. Final health check"
curl -sf "${BASE_URL}/api/v1/health" | jq .

echo
echo "Demo complete!"
