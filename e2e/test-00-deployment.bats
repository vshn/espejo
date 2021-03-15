#!/usr/bin/env bats

load "lib/utils"
load "lib/detik"
load "lib/custom"

DETIK_CLIENT_NAME="kubectl"
DETIK_CLIENT_NAMESPACE="espejo-system"
DEBUG_DETIK="true"

@test "verify the deployment" {
	# Remove traces of previous deployments from other tests
	kubectl delete namespace "$DETIK_CLIENT_NAMESPACE" --ignore-not-found
	kubectl create namespace "$DETIK_CLIENT_NAMESPACE"

	apply definitions/operator

	try "at most 10 times every 2s to find 1 pod named 'espejo-operator' with '.spec.containers[*].image' being '${E2E_IMAGE}'"
	try "at most 20 times every 2s to find 1 pod named 'espejo-operator' with 'status' being 'running'"
}
