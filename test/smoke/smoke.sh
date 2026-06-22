#!/bin/sh
# Smoke test for the runj containerd shim.  Drives the shim through the full
# container lifecycle (create, start, exec, kill, delete) with ctr against a
# running containerd daemon.  It is the per-step gate for the containerd
# migration; see docs/containerd-shim-migration.md.
#
# Runs as root, since ctr talks to the root-owned containerd socket.  Repeatable:
# leftover state from a prior run is removed before and after.
#
# Env overrides: RUNTIME, IMAGE, ID.
set -eu

RUNTIME="${RUNTIME:-wtf.sbk.runj.v1}"
IMAGE="${IMAGE:-public.ecr.aws/samuelkarp/freebsd:13.1-RELEASE}"
ID="${ID:-runj-smoke}"

fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }

cleanup() {
	ctr task kill -s SIGKILL "$ID" >/dev/null 2>&1 || true
	ctr task delete "$ID" >/dev/null 2>&1 || true
	ctr container delete "$ID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

[ "$(id -u)" -eq 0 ] || fail "must run as root"

if ! ctr version >/dev/null 2>&1; then
	service containerd onestart || fail "could not start containerd"
	for _ in 1 2 3 4 5 6 7 8 9 10; do
		ctr version >/dev/null 2>&1 && break
		sleep 1
	done
	ctr version >/dev/null 2>&1 || fail "containerd did not come up"
fi

ctr images ls -q | grep -q "$IMAGE" || ctr image pull "$IMAGE" || fail "image pull"

cleanup  # clear leftovers from an earlier run before starting

echo "== create + start =="
ctr run -d --runtime "$RUNTIME" "$IMAGE" "$ID" /bin/sh -c 'sleep 1000' \
	|| fail "create + start"

echo "== exec =="
out="$(ctr task exec --exec-id smoke-exec "$ID" /bin/sh -c 'echo EXEC_OK')" \
	|| fail "exec"
printf '%s\n' "$out" | grep -q EXEC_OK || fail "exec output missing EXEC_OK: $out"

echo "== kill =="
ctr task kill -s SIGKILL "$ID" || fail "kill"
for _ in 1 2 3 4 5; do
	ctr task ls 2>/dev/null | grep -q "$ID" || break
	sleep 1
done

echo "== delete =="
ctr task delete "$ID" || fail "task delete"
ctr container delete "$ID" || fail "container delete"

trap - EXIT
echo "SMOKE OK"
