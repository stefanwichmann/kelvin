#!/usr/bin/env bash
# Docker distribution smoke test (issue #139).
#
# Builds the image with the real Dockerfile and runs it the way a
# production deployment does — config.json bind-mounted as a single
# file — then asserts the failure modes that have actually shipped:
# the bind-mount crash loop (#135), the host-check lockout (#137),
# and SIGKILL on docker stop (#138).
#
# Requires Docker and network for the first base-image pull. Run it
# before committing changes to the Dockerfile, goreleaser config, web
# interface, or configuration I/O, and always before tagging a release.
set -euo pipefail

cd "$(dirname "$0")/.."

WORK=$(mktemp -d)
IMAGE=kelvin-smoke:local
CONTAINER=kelvin-smoke

cleanup() {
	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	docker rmi "$IMAGE" >/dev/null 2>&1 || true
	rm -rf "$WORK"
}
trap cleanup EXIT

fail() {
	echo "SMOKE FAIL: $*" >&2
	echo "--- container log:" >&2
	docker logs "$CONTAINER" 2>&1 | tail -20 >&2 || true
	exit 1
}

case "$(docker info --format '{{.Architecture}}')" in
aarch64 | arm64) GOARCH=arm64 ;;
x86_64) GOARCH=amd64 ;;
*) fail "unsupported docker architecture" ;;
esac

echo "==> building image"
mkdir -p "$WORK/ctx"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -o "$WORK/ctx/kelvin" .
cp Dockerfile "$WORK/ctx/"
cp -R gui "$WORK/ctx/gui"
docker build -q -t "$IMAGE" "$WORK/ctx" >/dev/null

echo "==> starting container (single-file config bind mount, version-1 config)"
# 192.0.2.1 is TEST-NET: the bridge connection may retry forever, which
# must not block the web interface or the assertions below.
cat >"$WORK/config.json" <<'EOF'
{"version":1,"bridge":{"ip":"192.0.2.1","username":"smoke"},"webinterface":{"enabled":true,"port":8080,"listenAddress":"0.0.0.0","password":"smoke"},"schedules":[{"name":"smoke","associatedDeviceIDs":[1]}]}
EOF
docker run -d --name "$CONTAINER" -p 127.0.0.1:0:8080 \
	-v "$WORK/config.json":/etc/opt/kelvin/config.json "$IMAGE" >/dev/null
PORT=$(docker port "$CONTAINER" 8080/tcp | head -1 | sed 's/.*://')

echo "==> waiting for /health"
up=""
for _ in $(seq 1 40); do
	if curl -fs -m 2 "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then
		up=yes
		break
	fi
	sleep 0.5
done
[ -n "$up" ] || fail "/health did not answer within 20s"

echo "==> asserting startup and readback"
if docker logs "$CONTAINER" 2>&1 | grep -qi "level=fatal"; then
	fail "fatal during startup (#135)"
fi
grep -q '"version": 2' "$WORK/config.json" ||
	fail "config not migrated through the file bind mount (#135)"

echo "==> asserting host check and authentication"
code=$(curl -s -o /dev/null -w '%{http_code}' -m 5 -H "Host: smokehost:8080" "http://127.0.0.1:$PORT/")
[ "$code" = "401" ] || fail "single-label host: got $code, want 401 (#137)"
code=$(curl -s -o /dev/null -w '%{http_code}' -m 5 -u ":smoke" "http://127.0.0.1:$PORT/")
[ "$code" = "200" ] || fail "authenticated GUI request: got $code, want 200 (gui templates)"

echo "==> asserting graceful shutdown"
start=$(date +%s)
docker stop -t 10 "$CONTAINER" >/dev/null
duration=$(($(date +%s) - start))
exitcode=$(docker inspect "$CONTAINER" --format '{{.State.ExitCode}}')
[ "$duration" -le 5 ] || fail "docker stop took ${duration}s — SIGTERM not reaching kelvin (#138)"
[ "$exitcode" != "137" ] || fail "exit code 137 — container was SIGKILLed (#138)"

echo "SMOKE OK (port $PORT, stop ${duration}s, exit $exitcode)"
