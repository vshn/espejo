#!/bin/bash

set -eo pipefail

test_dir=e2e-test

source "${test_dir}/functions.sh"

e2e-test-setup

WATCH_NAMESPACE=espejo start-espejo

sleep 5s
echo "- Creating sync config in watched namespace"
kubectl -n espejo apply -f "${test_dir}/syncconfig.yaml"
sleep 5s

echo "- Verifying that there is an object in another namespace"
kubectl get cm -n e2e-test | grep "espejo-e2e-test-data"

echo "--- TEST PASSED"

e2e-test-teardown
