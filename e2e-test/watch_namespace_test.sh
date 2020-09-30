#!/bin/bash

set -eo pipefail

test_dir=e2e-test

source "${test_dir}/functions.sh"

e2e-test-setup

WATCH_NAMESPACE=espejo start-espejo

sleep 5s
echo "- Creating sync config in an unwatched namespace"
kubectl -n default apply -f "${test_dir}/syncconfig.yaml"
sleep 5s

echo "- Verifying that there is no object in namespace"
kubectl get cm -n e2e-test 2>&1 | grep "No resources found in e2e-test namespace."

echo "--- TEST PASSED"

e2e-test-teardown
