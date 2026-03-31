#!/bin/sh
# Isolate Panel Docker Health Check
# Checks: 1) Panel HTTP health  2) Core process status (FATAL/BACKOFF = unhealthy)

# 1. Check panel HTTP endpoint
wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 2. Check active cores via supervisorctl
# Only fail if a core is in FATAL or BACKOFF state (i.e. crashed)
# STOPPED/EXITED = intentionally stopped — OK
# RUNNING/STARTING = working — OK
for prog in xray mihomo singbox; do
    status=$(supervisorctl status "$prog" 2>/dev/null)
    if echo "$status" | grep -qE "FATAL|BACKOFF"; then
        echo "UNHEALTHY: $prog is in $(echo "$status" | awk '{print $2}') state"
        exit 1
    fi
done

exit 0
