#!/usr/bin/env bash
# Seed the demo team with rich local data for console UI review.
# Requires API at API_URL (default http://localhost:18080) and demo key.
set -euo pipefail

API="${API_URL:-http://localhost:18080}"
KEY="${RAISIN_API_KEY:-ra_demo_00000000000000000000000000000000}"
WORKER="${WORKER_URL:-http://localhost:18081}"
UA="raisin-seed/0.1"
STAMP=$(date +%s)

auth=(-H "Authorization: Bearer $KEY" -H "User-Agent: $UA" -H "Content-Type: application/json")

# curl with retry on 429 (demo team is 60 req/s)
api() {
  local method="$1"; shift
  local path="$1"; shift
  local tries=0
  local out code
  while (( tries < 8 )); do
    out=$(mktemp)
    code=$(curl -s -o "$out" -w "%{http_code}" -X "$method" "$API$path" "${auth[@]}" "$@")
    if [[ "$code" == "429" ]]; then
      rm -f "$out"
      sleep 1
      tries=$((tries + 1))
      continue
    fi
    if [[ "$code" != 2* ]]; then
      echo "API $method $path -> $code" >&2
      cat "$out" >&2
      rm -f "$out"
      return 1
    fi
    cat "$out"
    rm -f "$out"
    return 0
  done
  echo "rate limited: $method $path" >&2
  return 1
}

echo "== health =="
curl -sf "$API/health" | grep -q ok

echo "== domains =="
ensure_domain() {
  local name="$1"
  local existing
  existing=$(curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/domains" \
    | python3 -c "import sys,json; 
d=json.load(sys.stdin).get('data') or []
for x in d:
  if x.get('name')=='$name':
    print(x['id']); break")
  if [[ -n "$existing" ]]; then
    echo "$existing"
    return
  fi
  local created
  created=$(curl -sf -X POST "$API/domains" "${auth[@]}" -d "{\"name\":\"$name\",\"region\":\"us-east-1\"}")
  local id
  id=$(echo "$created" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
  curl -sf -X POST "$API/domains/$id/verify" "${auth[@]}" >/dev/null || true
  echo "$id"
}
DOM_ACME_ID=$(ensure_domain "acme.test")
DOM_NEWS_ID=$(ensure_domain "news.acme.test")
echo "  acme.test=$DOM_ACME_ID news.acme.test=$DOM_NEWS_ID"
echo "== contact properties =="
for row in 'plan:string' 'company:string' 'seats:number' 'mrr:number' 'role:string'; do
  k="${row%%:*}"; t="${row##*:}"
  curl -sf -X POST "$API/contact-properties" "${auth[@]}" \
    -d "{\"key\":\"$k\",\"type\":\"$t\"}" >/dev/null || true
done

echo "== segments & topics =="
SEG_VIP=$(curl -sf -X POST "$API/segments" "${auth[@]}" -d '{"name":"VIP customers"}')
SEG_VIP_ID=$(echo "$SEG_VIP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
SEG_TRIAL=$(curl -sf -X POST "$API/segments" "${auth[@]}" -d '{"name":"Trial users"}')
SEG_TRIAL_ID=$(echo "$SEG_TRIAL" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
SEG_CHURN=$(curl -sf -X POST "$API/segments" "${auth[@]}" -d '{"name":"At risk"}')
SEG_CHURN_ID=$(echo "$SEG_CHURN" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

TP_PROD=$(curl -sf -X POST "$API/topics" "${auth[@]}" \
  -d '{"name":"Product updates","description":"Launches and changelog","default_subscription":"opt_out"}')
TP_PROD_ID=$(echo "$TP_PROD" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
TP_MKT=$(curl -sf -X POST "$API/topics" "${auth[@]}" \
  -d '{"name":"Marketing","description":"Campaigns and offers","default_subscription":"opt_in"}')
TP_MKT_ID=$(echo "$TP_MKT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
TP_BILL=$(curl -sf -X POST "$API/topics" "${auth[@]}" \
  -d '{"name":"Billing","description":"Invoices and receipts","default_subscription":"opt_out"}')
TP_BILL_ID=$(echo "$TP_BILL" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

echo "== contacts (24) =="
NAMES=(
  "Ada:Lovelace:ada@example.com:enterprise:Acme Robotics:120:4800:cto:vip"
  "Grace:Hopper:grace@example.com:pro:Navy Systems:40:1200:eng:vip"
  "Alan:Turing:alan@example.com:pro:Bletchley:25:900:eng:vip"
  "Katherine:Johnson:kathy@example.com:starter:NASA Lab:8:200:ops:trial"
  "Margaret:Hamilton:margaret@example.com:pro:Apollo Soft:60:1800:eng:vip"
  "Tim:Berners-Lee:tim@example.com:enterprise:Web Foundation:200:8000:founder:vip"
  "Barbara:Liskov:barbara@example.com:pro:MIT Labs:15:450:eng:trial"
  "Donald:Knuth:don@example.com:starter:Art of Code:3:90:founder:trial"
  "Radia:Perlman:radia@example.com:pro:Routing Co:22:660:eng:vip"
  "Lin:Huiyin:lin@example.com:starter:Studio North:5:150:design:trial"
  "Hedy:Lamarr:hedy@example.com:pro:Freq Hop:18:540:product:vip"
  "Claude:Shannon:claude@example.com:enterprise:Info Theory:90:3600:research:vip"
  "Annie:Easley:annie@example.com:starter:Rocket Fuel:6:180:ops:trial"
  "John:von Neumann:john@example.com:enterprise:IAS Compute:150:6000:research:vip"
  "Dorothy:Vaughan:dorothy@example.com:pro:Langley:30:900:eng:vip"
  "Edsger:Dijkstra:edsger@example.com:pro:Graph Walk:12:360:eng:trial"
  "Frances:Allen:fran@example.com:starter:Compiler Co:4:120:eng:trial"
  "Dennis:Ritchie:dennis@example.com:pro:Unix Desk:20:600:eng:vip"
  "Ken:Thompson:ken@example.com:pro:Bell Labs:20:600:eng:vip"
  "Sophie:Wilson:sophie@example.com:enterprise:ARM Design:110:4400:hw:vip"
  "Vint:Cerf:vint@example.com:enterprise:Packet Net:80:3200:founder:vip"
  "Bob:Kahn:bob@example.com:pro:Protocol Lab:35:1050:eng:vip"
  "Evelyn:Boyd:evelyn@example.com:starter:Math Works:7:210:research:trial"
  "Chien:Shiung:wu@example.com:pro:Parity Lab:28:840:research:vip"
)

CONTACT_IDS=()
n=0
for row in "${NAMES[@]}"; do
  IFS=':' read -r first last email plan company seats mrr role bucket <<<"$row"
  CT=$(api POST /contacts -d "{
    \"email\":\"$email\",
    \"first_name\":\"$first\",
    \"last_name\":\"$last\",
    \"properties\":{\"plan\":\"$plan\",\"company\":\"$company\",\"seats\":$seats,\"mrr\":$mrr,\"role\":\"$role\"}
  }")
  CT_ID=$(echo "$CT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
  CONTACT_IDS+=("$CT_ID")
  case "$bucket" in
    vip) SEG="$SEG_VIP_ID" ;;
    trial) SEG="$SEG_TRIAL_ID" ;;
    *) SEG="$SEG_TRIAL_ID" ;;
  esac
  api POST "/contacts/$CT_ID/segments/$SEG" -d '{}' >/dev/null || true
  api PUT "/contacts/$CT_ID/topics/$TP_PROD_ID" -d '{"subscribed":true}' >/dev/null || true
  if [[ "$bucket" == "vip" ]]; then
    api PUT "/contacts/$CT_ID/topics/$TP_MKT_ID" -d '{"subscribed":true}' >/dev/null || true
  else
    api PUT "/contacts/$CT_ID/topics/$TP_MKT_ID" -d '{"subscribed":false}' >/dev/null || true
  fi
  api PUT "/contacts/$CT_ID/topics/$TP_BILL_ID" -d '{"subscribed":true}' >/dev/null || true
  n=$((n + 1))
  # stay under 60 req/s demo rate limit
  if (( n % 8 == 0 )); then
    sleep 1
  fi
done
# a few at-risk
for i in 0 1 2; do
  curl -sf -X POST "$API/contacts/${CONTACT_IDS[$i]}/segments/$SEG_CHURN_ID" "${auth[@]}" -d '{}' >/dev/null || true
done

echo "== suppressions =="
curl -sf -X POST "$API/suppressions" "${auth[@]}" \
  -d '{"email":"bounce-hard@example.com","reason":"bounce"}' >/dev/null || true
curl -sf -X POST "$API/suppressions" "${auth[@]}" \
  -d '{"email":"complaint@example.com","reason":"complaint"}' >/dev/null || true
curl -sf -X POST "$API/suppressions/batch" "${auth[@]}" \
  -d '{"emails":["old-unsub-1@example.com","old-unsub-2@example.com"],"reason":"unsubscribe"}' >/dev/null || true

echo "== templates =="
TPL_WELCOME=$(curl -sf -X POST "$API/templates" "${auth[@]}" -d @- <<'EOF'
{"name":"Welcome series","subject":"Welcome, {{first_name}}","html":"<html><body><h1>Welcome {{first_name}}</h1><p>Thanks for joining {{company}}.</p><p><a href=\"https://raisin.run\">Open dashboard</a></p></body></html>","text":"Welcome {{first_name}}"}
EOF
)
TPL_WELCOME_ID=$(echo "$TPL_WELCOME" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/templates/$TPL_WELCOME_ID/publish" "${auth[@]}" -d '{}' >/dev/null || true

TPL_DIGEST=$(curl -sf -X POST "$API/templates" "${auth[@]}" -d @- <<'EOF'
{"name":"Weekly digest","subject":"Your week at {{company}}","html":"<html><body><p>Hi {{first_name}}, here is this week.</p><ul><li>Ship notes</li><li>Usage tips</li></ul></body></html>"}
EOF
)
TPL_DIGEST_ID=$(echo "$TPL_DIGEST" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/templates/$TPL_DIGEST_ID/publish" "${auth[@]}" -d '{}' >/dev/null || true

TPL_DRAFT=$(curl -sf -X POST "$API/templates" "${auth[@]}" -d @- <<'EOF'
{"name":"Re-engagement (draft)","subject":"We miss you, {{first_name}}","html":"<p>Come back soon.</p>"}
EOF
)
echo "$TPL_DRAFT" | grep -q '"id"'

echo "== webhooks =="
WH=$(curl -sf -X POST "$API/webhooks" "${auth[@]}" -d '{
  "endpoint":"https://example.com/hooks/raisin",
  "events":["email.sent","email.delivered","email.bounced","email.opened","email.clicked","email.complained"]
}')
WH_ID=$(echo "$WH" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/webhooks" "${auth[@]}" -d '{
  "endpoint":"https://hooks.example.dev/inbox",
  "events":["email.bounced","email.complained"]
}' >/dev/null || true

echo "== emails + events =="
EMAIL_IDS=()
SUBJECTS=(
  "Welcome to Raisin"
  "Your invoice is ready"
  "Product update: topics"
  "Security notice"
  "Weekly digest"
  "Invite your team"
  "Broadcast preview"
  "Password reset"
)
for i in "${!SUBJECTS[@]}"; do
  to="seed-$i-$STAMP@example.com"
  if (( i < ${#NAMES[@]} )); then
    IFS=':' read -r _ _ to _ <<<"${NAMES[$i]}"
  fi
  RESP=$(curl -sf -X POST "$API/emails" "${auth[@]}" \
    -H "Idempotency-Key: seed-$STAMP-$i" \
    -d @- <<EOF
{"from":"Acme <hello@acme.test>","to":["$to"],"subject":"${SUBJECTS[$i]}","html":"<html><body><p>Hello from seed #$i</p><p><a href=\"https://raisin.run/docs\">Docs</a></p></body></html>","tags":{"campaign":"seed","n":"$i"}}
EOF
)
  EID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
  EMAIL_IDS+=("$EID")
done

# wait for delivery then synthesize engagement for overview metrics
sleep 1
for i in "${!EMAIL_IDS[@]}"; do
  EID="${EMAIL_IDS[$i]}"
  for _ in $(seq 1 10); do
    ST=$(curl -sf -H "Authorization: Bearer $KEY" -H "User-Agent: $UA" "$API/emails/$EID" \
      | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || true)
    [[ "$ST" == "delivered" || "$ST" == "sent" ]] && break
    sleep 0.3
  done
  if (( i % 2 == 0 )); then
    curl -sf -X POST "$WORKER/test/events/$EID?type=email.opened" >/dev/null || true
  fi
  if (( i % 3 == 0 )); then
    curl -sf -X POST "$WORKER/test/events/$EID?type=email.clicked" >/dev/null || true
  fi
  if (( i == 7 )); then
    curl -sf -X POST "$WORKER/test/events/$EID?type=email.bounced" >/dev/null || true
  fi
  sleep 0.2
done

echo "== broadcasts =="
BC_DRAFT=$(curl -sf -X POST "$API/broadcasts" "${auth[@]}" -d "{
  \"name\":\"March product update\",
  \"from\":\"Acme <hello@acme.test>\",
  \"subject\":\"What is new in Raisin\",
  \"html\":\"<p>Topics, properties, and scheduled sends.</p>\",
  \"topic_id\":\"$TP_PROD_ID\",
  \"segment_id\":\"$SEG_VIP_ID\"
}")
BC_DRAFT_ID=$(echo "$BC_DRAFT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

BC_SENT=$(curl -sf -X POST "$API/broadcasts" "${auth[@]}" -d "{
  \"name\":\"VIP thank you\",
  \"from\":\"Acme <hello@acme.test>\",
  \"subject\":\"Thanks for being a VIP\",
  \"html\":\"<p>We appreciate you.</p>\",
  \"segment_id\":\"$SEG_VIP_ID\"
}")
BC_SENT_ID=$(echo "$BC_SENT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/broadcasts/$BC_SENT_ID/send" "${auth[@]}" -d '{}' >/dev/null || true

SCHED_AT=$(python3 -c "from datetime import datetime,timedelta,timezone; print((datetime.now(timezone.utc)+timedelta(days=2)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
BC_SCHED=$(curl -sf -X POST "$API/broadcasts" "${auth[@]}" -d "{
  \"name\":\"Scheduled newsletter\",
  \"from\":\"News <hello@news.acme.test>\",
  \"subject\":\"This week in Acme\",
  \"html\":\"<p>Newsletter body</p>\",
  \"topic_id\":\"$TP_MKT_ID\",
  \"scheduled_at\":\"$SCHED_AT\"
}")
BC_SCHED_ID=$(echo "$BC_SCHED" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X POST "$API/broadcasts/$BC_SCHED_ID/send" "${auth[@]}" -d '{}' >/dev/null || true

echo "== automations =="
AU=$(curl -sf -X POST "$API/automations" "${auth[@]}" -d '{
  "name":"Welcome on contact.created",
  "trigger_type":"contact.created",
  "steps":[
    {"type":"wait","config":{"seconds":60}},
    {"type":"send_email","config":{"from":"Acme <hello@acme.test>","subject":"Welcome aboard","html":"<p>Hi there</p>"}}
  ]
}')
AU_ID=$(echo "$AU" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
curl -sf -X PATCH "$API/automations/$AU_ID" "${auth[@]}" -d '{"enabled":true}' >/dev/null || true

AU2=$(curl -sf -X POST "$API/automations" "${auth[@]}" -d '{
  "name":"Bounce follow-up",
  "trigger_type":"email.bounced",
  "steps":[{"type":"wait","config":{"seconds":30}}]
}')
echo "$AU2" | grep -q '"id"' || true

echo "== ip pools =="
curl -sf -X POST "$API/ip-pools" "${auth[@]}" \
  -d '{"name":"Transactional pool"}' >/dev/null || true
curl -sf -X POST "$API/ip-pools" "${auth[@]}" \
  -d '{"name":"Marketing warmup"}' >/dev/null || true

echo "== oauth apps =="
curl -sf -X POST "$API/oauth/apps" "${auth[@]}" -d '{
  "name":"Acme Integrations",
  "redirect_uris":["http://localhost:3001/oauth/callback","https://app.acme.test/oauth/callback"]
}' >/dev/null || true

echo "== api keys =="
curl -sf -X POST "$API/api-keys" "${auth[@]}" \
  -d '{"name":"CI sending","permission":"sending_access"}' >/dev/null || true
curl -sf -X POST "$API/api-keys" "${auth[@]}" \
  -d '{"name":"Staging full","permission":"full_access"}' >/dev/null || true

# give broadcasts a moment
sleep 2

echo
echo "Seed complete."
echo "  API:     $API"
echo "  Worker:  $WORKER"
echo "  Mailpit: ${MAILPIT_URL:-http://localhost:8026}"
echo "  Demo key: $KEY"
echo "  Open console → Continue with seeded demo team"
echo "  Sample IDs: domain=$DOM_ACME_ID webhook=$WH_ID broadcast_draft=$BC_DRAFT_ID"
