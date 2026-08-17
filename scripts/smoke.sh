#!/usr/bin/env bash
set -euo pipefail

API="${API_URL:-http://localhost:18080}"
KEY="${RAISIN_API_KEY:-ra_demo_00000000000000000000000000000000}"
UA="raisin-smoke/0.1"
STAMP=$(date +%s)

echo "== health =="
curl -sf "$API/health" | grep -q ok

echo "== send email =="
RESP=$(curl -sf -D /tmp/raisin-smoke-headers -X POST "$API/emails" \
  -H "Authorization: Bearer $KEY" \
  -H "User-Agent: $UA" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: smoke-$STAMP" \
  -d '{"from":"Acme <hello@acme.test>","to":["smoke@example.com"],"subject":"Smoke","html":"<p>ok</p>","attachments":[{"filename":"note.txt","content_type":"text/plain","content":"aGVsbG8="}]}')
echo "$RESP" | grep -q '"status"'
EMAIL_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
grep -qi 'X-RateLimit-Limit' /tmp/raisin-smoke-headers

echo "== wait for delivery =="
for i in $(seq 1 20); do
  ST=$(curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/$EMAIL_ID" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
  if [[ "$ST" == "delivered" || "$ST" == "sent" ]]; then
    echo "status=$ST"
    break
  fi
  sleep 0.5
done

echo "== attachments =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/$EMAIL_ID/attachments" | grep -q note.txt

echo "== events =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/$EMAIL_ID/events" | grep -q email.sent

echo "== domain create + verify =="
DOM=$(curl -sf -X POST "$API/domains" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-$STAMP.example.com\"}")
echo "$DOM" | grep -q "smoke-$STAMP.example.com"
DOM_ID=$(echo "$DOM" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/domains/$DOM_ID/verify" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" \
  | grep -q '"status":"verified"'

echo "== team test_mode =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/team" | grep -q test_mode
curl -sf -X PATCH "$API/team" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{"test_mode":true}' | grep -q '"test_mode":true'

echo "== webhook create + events =="
WH=$(curl -sf -X POST "$API/webhooks" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{"endpoint":"https://example.com/hooks/smoke","events":["email.sent","email.delivered"]}')
WH_ID=$(echo "$WH" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/webhooks/$WH_ID/events" | grep -q '"data"'

echo "== inbound direct =="
curl -sf -X POST "$API/inbound/ses" \
  -H "Content-Type: application/json" -H "User-Agent: $UA" \
  -d '{"team_id":"00000000-0000-0000-0000-000000000001","from":"someone@elsewhere.com","to":["inbox@acme.test"],"subject":"Inbound","text":"hi"}' \
  | grep -q '"from"'

echo "== received =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/received" | grep -q '"data"'

echo "== provision =="
curl -sf -X POST "$API/console/provision" \
  -H "Content-Type: application/json" -H "User-Agent: $UA" \
  -d "{\"secret\":\"${JWT_SECRET:-dev-jwt-secret-change-me-in-production}\",\"email\":\"user-$STAMP@raisin.run\",\"name\":\"Smoke User\"}" \
  | grep -q '"token"'

echo "OK"
