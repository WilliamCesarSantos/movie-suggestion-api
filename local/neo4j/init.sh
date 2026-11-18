#!/bin/bash
# Sourced by the Neo4j entrypoint via EXTENSION_SCRIPT, before Neo4j starts.
# Runs in a background subshell so it never blocks or aborts the server startup:
# waits for the database to accept connections, then applies init.cypher.

(
  INIT_FILE="${NEO4J_INIT_FILE:-/init/init.cypher}"
  MAX_WAIT_SECONDS="${NEO4J_INIT_TIMEOUT_SECONDS:-300}"
  RETRY_INTERVAL_SECONDS="${NEO4J_INIT_RETRY_SECONDS:-30}"
  NEO4J_USER="${NEO4J_AUTH%%/*}"
  NEO4J_PASS="${NEO4J_AUTH#*/}"

  log() { echo "[neo4j-init] $(date -u +%Y-%m-%dT%H:%M:%SZ) $*"; }

  deadline=$((SECONDS + MAX_WAIT_SECONDS))
  attempt=0

  log "waiting for Neo4j (timeout ${MAX_WAIT_SECONDS}s, retry every ${RETRY_INTERVAL_SECONDS}s)"
  while true; do
    attempt=$((attempt + 1))
    if timeout 10 cypher-shell -u "$NEO4J_USER" -p "$NEO4J_PASS" "RETURN 1" >/dev/null 2>&1; then
      log "Neo4j online after ${attempt} attempt(s)"
      break
    fi

    remaining=$((deadline - SECONDS))
    if [ "$remaining" -le 0 ]; then
      log "ERROR: Neo4j not online after ${MAX_WAIT_SECONDS}s (${attempt} attempts), giving up"
      exit 1
    fi

    sleep_for=$((remaining < RETRY_INTERVAL_SECONDS ? remaining : RETRY_INTERVAL_SECONDS))
    log "attempt ${attempt} failed, retrying in ${sleep_for}s (${remaining}s left)"
    sleep "$sleep_for"
  done

  if cypher-shell -u "$NEO4J_USER" -p "$NEO4J_PASS" -f "$INIT_FILE"; then
    log "applied ${INIT_FILE}"
  else
    log "ERROR: failed to apply ${INIT_FILE}"
    exit 1
  fi
) &
