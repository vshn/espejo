#!/usr/bin/env bats

load "lib/utils"
load "lib/detik"
load "lib/custom"

# shellcheck disable=SC2034
DETIK_CLIENT_NAME="kubectl"
# shellcheck disable=SC2034
DETIK_CLIENT_NAMESPACE="espejo-system"
# shellcheck disable=SC2034
DEBUG_DETIK="true"

@test "Given a SyncConfig manifest, When creating it, Then expect synced items" {
	given_running_operator

	apply definitions/syncconfig
	try "at most 10 times every 1s to get configmap named 'espejo-e2e-test-data' and verify that '.data.KEY' is 'espejo-system'"
}
