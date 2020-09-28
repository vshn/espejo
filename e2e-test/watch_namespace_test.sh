#!/bin/bash

set -eo pipefail

test_dir=e2e-test

source ${test_dir}/functions.sh

e2e-test-setup

WATCH_NAMESPACE=espejo go run main.go &
PID=$!
echo "Started espejo with PID=${PID}"

sleep 5s
echo "Creating sync config in an unwatched namespace"
kubectl apply -f ${test_dir}/syncconfig.yaml
sleep 5s

kubectl get cm -n e2e-test

kill ${PID}

e2e-test-teardown

