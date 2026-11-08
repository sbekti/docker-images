#!/bin/sh
set -eu
image=${1:-rsyslog:test}
work=$(mktemp -d)
container="rsyslog-test-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT
mkdir "$work/logs"
chmod 777 "$work/logs"
cat > "$work/rsyslog.conf" <<'CONF'
global(workDirectory="/run/rsyslog")
module(load="imudp")
template(name="Daily" type="string" string="/logs/%timegenerated:::date-year%-%timegenerated:::date-month%-%timegenerated:::date-day%.log")
template(name="Line" type="string" string="%timegenerated:::date-rfc3339,date-utc% sender=%fromhost-ip% host=%hostname% %syslogtag%%msg%\n")
action(type="omfile" dynaFile="Daily" template="Line" fileCreateMode="0640")
input(type="imudp" port="15514")
CONF
chmod 644 "$work/rsyslog.conf"
docker run --rm --entrypoint sh "$image" -ec '
    test "$(id -u)" = 10001
    test "$(apk version -t "$(cat /etc/alpine-release)" 3.24.4)" != "<"
    ! apk info -e openssh-server
    ! command -v sshd
    apk list --installed musl rsyslog
'
docker run -d --name "$container" --read-only --cap-drop ALL \
    --security-opt no-new-privileges \
    --tmpfs /run/rsyslog:uid=10001,gid=10001,mode=0700,size=16m \
    -e TZ=UTC -v "$work/rsyslog.conf:/etc/rsyslog.conf:ro" \
    -v "$work/logs:/logs" "$image" >/dev/null
docker exec "$container" rsyslogd -N1 -f /etc/rsyslog.conf >/dev/null 2>&1
for attempt in 1 2 3 4 5; do
    docker exec "$container" sh -c "printf '<14>Jan  1 00:00:00 example.test test: daily-file-probe\n' | nc -u -w 1 127.0.0.1 15514"
    if test -s "$work/logs/$(date -u +%F).log"; then break; fi
    sleep 1
done
docker exec "$container" grep -q 'sender=127.0.0.1 host=example.test.*daily-file-probe' "/logs/$(date -u +%F).log"
# Exercise the same simple retention operation used by a daily cleanup job.
docker exec "$container" sh -ec '
    touch -d @1 /logs/1970-01-01.log
    find /logs -maxdepth 1 -type f -name "????-??-??.log" -mtime +29 -delete
    test ! -e /logs/1970-01-01.log
    test -s "/logs/$(date -u +%F).log"
'
printf '%s\n' 'PASS non-root UDP reception, receipt-date files, retention and SSH-free packages'
