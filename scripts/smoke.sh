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

echo "== inbound rejects unsigned SNS =="
CODE=$(curl -s -o /tmp/raisin-sns-reject.json -w "%{http_code}" -X POST "$API/inbound/ses" \
  -H "Content-Type: application/json" -H "User-Agent: $UA" \
  -d '{"Type":"Notification","Message":"{}","MessageId":"x","TopicArn":"arn:aws:sns:us-east-1:1:t","Timestamp":"2020-01-01T00:00:00.000Z"}')
if [[ "$CODE" != "403" ]]; then
  echo "expected 403 for unsigned SNS, got $CODE" >&2
  cat /tmp/raisin-sns-reject.json >&2
  exit 1
fi
grep -q invalid_sns_signature /tmp/raisin-sns-reject.json

echo "== received =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/received" | grep -q '"data"'

echo "== contact =="
CT=$(curl -sf -X POST "$API/contacts" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"email\":\"smoke-$STAMP@example.com\",\"first_name\":\"Smoke\",\"last_name\":\"Test\"}")
echo "$CT" | grep -q "smoke-$STAMP@example.com"
CT_ID=$(echo "$CT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/contacts/$CT_ID" | grep -q first_name

echo "== api keys list =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/api-keys" | grep -q '"data"'

echo "== domain claim + receiving =="
curl -sf -X POST "$API/domains/$DOM_ID/claim" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" | grep -q Claim
curl -sf -X POST "$API/domains/$DOM_ID/claim/confirm" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" | grep -q claimed_at
curl -sf -X POST "$API/domains/$DOM_ID/receiving" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{"enabled":true}' | grep -q '"receiving_enabled":true'

echo "== domain delete =="
curl -sf -X DELETE "$API/domains/$DOM_ID" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" \
  | grep -q '"deleted":true'

echo "== logs =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/logs" | grep -q '"data"'

echo "== suppressions =="
SP=$(curl -sf -X POST "$API/suppressions" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"email\":\"blocked-$STAMP@example.com\",\"reason\":\"manual\"}")
SP_ID=$(echo "$SP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/suppressions" | grep -q "blocked-$STAMP@example.com"
curl -sf -X DELETE "$API/suppressions/$SP_ID" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" \
  | grep -q '"deleted":true'

echo "== template =="
TP=$(curl -sf -X POST "$API/templates" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-$STAMP\",\"subject\":\"Hi\",\"html\":\"<p>hi</p>\"}")
TP_ID=$(echo "$TP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/templates/$TP_ID/publish" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" \
  | grep -q '"status":"published"'

echo "== broadcast =="
BC=$(curl -sf -X POST "$API/broadcasts" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"from\":\"Acme <hello@acme.test>\",\"subject\":\"Smoke broadcast\",\"html\":\"<p>bc</p>\",\"name\":\"smoke-$STAMP\"}")
BC_ID=$(echo "$BC" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/broadcasts/$BC_ID/send" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{}' | grep -q '"status"'
curl -sf -X DELETE "$API/broadcasts/$BC_ID" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" \
  | grep -q '"deleted":true'

echo "== automations =="
AU=$(curl -sf -X POST "$API/automations" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-$STAMP\",\"trigger_type\":\"contact.created\",\"steps\":[{\"type\":\"wait\",\"config\":{\"seconds\":1}}]}")
AU_ID=$(echo "$AU" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X PATCH "$API/automations/$AU_ID" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{"enabled":true}' | grep -q '"enabled":true'
AU_CT=$(curl -sf -X POST "$API/contacts" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"email\":\"auto-$STAMP@example.com\",\"first_name\":\"Auto\"}")
echo "$AU_CT" | grep -q "auto-$STAMP@example.com"
RUN_OK=0
for i in $(seq 1 20); do
  RUNS=$(curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/automations/$AU_ID/runs" || true)
  if echo "$RUNS" | grep -q '"status"'; then
    RUN_OK=1
    break
  fi
  sleep 0.5
done
if [[ "$RUN_OK" != "1" ]]; then
  echo "automation run not created" >&2
  exit 1
fi

echo "== ip pools =="
POOL=$(curl -sf -X POST "$API/ip-pools" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-$STAMP\",\"region\":\"us-east-1\"}")
echo "$POOL" | grep -q '"warmup"'
POOL_ID=$(echo "$POOL" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
DBURL="${DATABASE_URL:-postgres://raisin:raisin@localhost:5433/raisin?sslmode=disable}"
if command -v psql >/dev/null 2>&1; then
  psql "$DBURL" -q -c "UPDATE warmup_schedules SET last_reset_at = now() - interval '2 days', sent_today = 5 WHERE pool_id = '$POOL_ID'"
  TICK=$(curl -sf -X POST "$API/ip-pools/$POOL_ID/warmup/tick" \
    -H "Authorization: Bearer $KEY" -H "User-Agent: $UA")
  echo "$TICK" | python3 -c "import sys,json; w=json.load(sys.stdin).get('warmup') or {}; assert w.get('day_index',0)>=1 and w.get('sent_today')==0, w"
fi

echo "== regions =="
curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/domains/regions" | grep -q us-east-1

echo "== oauth app =="
OA=$(curl -sf -X POST "$API/oauth/apps" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"name\":\"smoke-$STAMP\",\"redirect_uris\":[\"http://localhost:9999/cb\"],\"scopes\":[\"emails:read\",\"emails:send\"]}")
echo "$OA" | grep -q client_id
CLIENT_ID=$(echo "$OA" | python3 -c "import sys,json; print(json.load(sys.stdin)['client_id'])")
CLIENT_SECRET=$(echo "$OA" | python3 -c "import sys,json; print(json.load(sys.stdin)['client_secret'])")
OA_ID=$(echo "$OA" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf "$API/oauth/apps/public?client_id=$CLIENT_ID" -H "User-Agent: $UA" | grep -q "$CLIENT_ID"

# Authorize with console provision token
PROV=$(curl -sf -X POST "$API/console/provision" \
  -H "Content-Type: application/json" -H "User-Agent: $UA" \
  -d "{\"secret\":\"${JWT_SECRET:-dev-jwt-secret-change-me-in-production}\",\"email\":\"oauth-$STAMP@raisin.run\",\"name\":\"OAuth Smoke\"}")
TEAM_TOK=$(echo "$PROV" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
CODE=$(curl -sf -X POST "$API/oauth/authorize" \
  -H "X-Team-Token: $TEAM_TOK" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$CLIENT_ID\",\"redirect_uri\":\"http://localhost:9999/cb\",\"scopes\":[\"emails:read\"]}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['code'])")
TOK=$(curl -sf -X POST "$API/oauth/token" \
  -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"grant_type\":\"authorization_code\",\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\",\"code\":\"$CODE\",\"redirect_uri\":\"http://localhost:9999/cb\"}")
AT=$(echo "$TOK" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
RT=$(echo "$TOK" | python3 -c "import sys,json; print(json.load(sys.stdin)['refresh_token'])")
curl -sf -H "Authorization: Bearer $AT" -H "User-Agent: $UA" "$API/emails" | grep -q '"data"'
# emails:read token should not be allowed to send
if curl -sf -X POST "$API/emails" \
  -H "Authorization: Bearer $AT" -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d '{"from":"x@y.z","to":["a@b.c"],"subject":"no","html":"<p>x</p>"}' >/dev/null 2>&1; then
  echo "oauth scope check failed: send should be forbidden" >&2
  exit 1
fi
REF=$(curl -sf -X POST "$API/oauth/token" \
  -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"grant_type\":\"refresh_token\",\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\",\"refresh_token\":\"$RT\"}")
AT2=$(echo "$REF" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
curl -sf -H "Authorization: Bearer $AT2" -H "User-Agent: $UA" "$API/emails" | grep -q '"data"'
# old refresh token must not work twice
if curl -sf -X POST "$API/oauth/token" \
  -H "User-Agent: $UA" -H "Content-Type: application/json" \
  -d "{\"grant_type\":\"refresh_token\",\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\",\"refresh_token\":\"$RT\"}" >/dev/null 2>&1; then
  echo "oauth refresh reuse should fail" >&2
  exit 1
fi
curl -sf -X DELETE "$API/oauth/apps/$OA_ID" \
  -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" | grep -q '"deleted":true'

echo "== provision =="
curl -sf -X POST "$API/console/provision" \
  -H "Content-Type: application/json" -H "User-Agent: $UA" \
  -d "{\"secret\":\"${JWT_SECRET:-dev-jwt-secret-change-me-in-production}\",\"email\":\"user-$STAMP@raisin.run\",\"name\":\"Smoke User\"}" \
  | grep -q '"token"'

echo "OK"
