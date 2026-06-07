#!/usr/bin/env bash
# test.sh — Maradhi API end-to-end tests
# Usage: bash test.sh
set -euo pipefail

BASE="http://localhost:8080"
ADMIN_SECRET="${ADMIN_SECRET:-}"

GREEN='\033[0;32m'; RED='\033[0;31m'; CYAN='\033[0;36m'; DIM='\033[2m'; NC='\033[0m'
pass()    { echo -e "${GREEN}  ✓ $1${NC}"; }
fail()    { echo -e "${RED}  ✗ $1${NC}"; exit 1; }
section() { echo -e "\n${CYAN}── $1 ──────────────────────────────────${NC}"; }

# Call API, pretty print, return body
# In test.sh, change the api() function — display to stderr, return to stdout
api() {
  local method=$1 path=$2; shift 2
  local args=(-s -X "$method" "$BASE$path" -H "Content-Type: application/json" "$@")
  [ -n "${TOKEN:-}" ] && args+=(-H "Authorization: Bearer $TOKEN")
  local resp; resp=$(curl "${args[@]}")
  echo "$resp" | jq . 2>/dev/null >&2 || echo "$resp" >&2  # display → stderr
  echo "$resp"                                               # return → stdout
}
# ── Server up? ────────────────────────────────────────────────────────────────
section "Health"
if ! curl -sf "$BASE/health" > /dev/null; then
  fail "Server not running — run: make run"
fi
api GET /health > /dev/null
pass "Server is up"

# ── Check ADMIN_SECRET ────────────────────────────────────────────────────────
if [ -z "$ADMIN_SECRET" ]; then
  echo ""
  echo "  Set your admin secret first:"
  echo "    export ADMIN_SECRET=\$(grep ADMIN_SECRET .env | cut -d= -f2)"
  echo ""
  exit 1
fi

# ── Register test user ────────────────────────────────────────────────────────
section "Register user"
REG=$(api POST /auth/register \
  -H "X-Admin-Secret: $ADMIN_SECRET" \
  -d '{"username":"aadhi","password":"test1234!","is_admin":true}')

TOKEN=$(echo "$REG" | jq -r '.token // empty')
USER_ID=$(echo "$REG" | jq -r '.user_id // empty')

if [ -z "$TOKEN" ]; then
  echo "  User may already exist — trying login instead..."
  section "Login"
  LOGIN=$(api POST /auth/login -d '{"username":"aadhi","password":"yourpassword"}')
  TOKEN=$(echo "$LOGIN" | jq -r '.token // empty')
  USER_ID=$(echo "$LOGIN" | jq -r '.user_id // empty')
fi

[ -n "$TOKEN" ] && pass "Got token for user: $USER_ID" || fail "Auth failed"

# ── Me ────────────────────────────────────────────────────────────────────────
section "GET /auth/me"
api GET /auth/me > /dev/null
pass "Me OK"

# ── Tags ──────────────────────────────────────────────────────────────────────
section "Tags"
TAG=$(api POST /api/v1/tags -d '{"name":"Work","color":"#111111"}')
TAG_ID=$(echo "$TAG" | jq -r '.data.id // empty')
[ -n "$TAG_ID" ] && pass "Tag created: $TAG_ID" || fail "Tag creation failed"
api GET /api/v1/tags > /dev/null && pass "Tags listed"

# ── Tasks ─────────────────────────────────────────────────────────────────────
section "Tasks"
TASK=$(api POST /api/v1/tasks -d "{
  \"title\":\"Review design system\",
  \"priority\":\"high\",
  \"tag_ids\":[\"$TAG_ID\"]
}")
TASK_ID=$(echo "$TASK" | jq -r '.data.id // empty')
[ -n "$TASK_ID" ] && pass "Task created: $TASK_ID" || fail "Task creation failed"

api GET /api/v1/tasks > /dev/null                && pass "Tasks listed"
api GET "/api/v1/tasks?priority=high" > /dev/null && pass "Tasks filtered"
api GET "/api/v1/tasks/$TASK_ID" > /dev/null      && pass "Task by ID"
api PATCH "/api/v1/tasks/$TASK_ID" \
  -d '{"status":"done"}' > /dev/null             && pass "Task updated"

# ── Bucket list ───────────────────────────────────────────────────────────────
section "Bucket List"
BKT=$(api POST /api/v1/bucket \
  -d '{"title":"Visit Japan","category":"travel"}')
BKT_ID=$(echo "$BKT" | jq -r '.data.id // empty')
[ -n "$BKT_ID" ] && pass "Bucket item created" || fail "Bucket creation failed"
api PATCH "/api/v1/bucket/$BKT_ID" -d '{"is_done":true}' > /dev/null && pass "Marked done"

# ── Notes ─────────────────────────────────────────────────────────────────────
section "Notes"
NOTE=$(api POST /api/v1/notes \
  -d '{"title":"Design audit","content":"Check all tokens","tag_label":"work"}')
NOTE_ID=$(echo "$NOTE" | jq -r '.data.id // empty')
[ -n "$NOTE_ID" ] && pass "Note created" || fail "Note creation failed"
api GET "/api/v1/notes?search=design" > /dev/null && pass "Notes searched"

# ── Habits ────────────────────────────────────────────────────────────────────
section "Habits"
HABIT=$(api POST /api/v1/habits -d '{"name":"Read"}')
HABIT_ID=$(echo "$HABIT" | jq -r '.data.id // empty')
[ -n "$HABIT_ID" ] && pass "Habit created" || fail "Habit creation failed"
TODAY=$(date +%Y-%m-%d)
api POST "/api/v1/habits/$HABIT_ID/log" \
  -d "{\"date\":\"$TODAY\",\"done\":true}" > /dev/null && pass "Habit logged"

# ── Focus ─────────────────────────────────────────────────────────────────────
section "Focus Sessions"
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
api POST /api/v1/focus -d "{
  \"duration_minutes\":25,
  \"task_note\":\"Deep work\",
  \"started_at\":\"$NOW\",
  \"ended_at\":\"$NOW\"
}" > /dev/null && pass "Focus session saved"

# ── Mood ──────────────────────────────────────────────────────────────────────
section "Mood"
api POST /api/v1/mood \
  -d "{\"mood\":\"good\",\"logged_at\":\"$NOW\"}" > /dev/null && pass "Mood logged"
api GET /api/v1/mood > /dev/null && pass "Mood listed"

# ── Cleanup ───────────────────────────────────────────────────────────────────
section "Cleanup"
api DELETE "/api/v1/tasks/$TASK_ID"   > /dev/null && pass "Task deleted"
api DELETE "/api/v1/tags/$TAG_ID"     > /dev/null && pass "Tag deleted"
api DELETE "/api/v1/bucket/$BKT_ID"   > /dev/null && pass "Bucket item deleted"
api DELETE "/api/v1/notes/$NOTE_ID"   > /dev/null && pass "Note deleted"
api DELETE "/api/v1/habits/$HABIT_ID" > /dev/null && pass "Habit deleted"

echo ""
echo -e "${GREEN}  All tests passed ✓${NC}"
echo ""
