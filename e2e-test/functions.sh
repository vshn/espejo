#!/bin/bash

PID=0
function finish() {
  if [[ ${PID} != 0 ]]; then
    kill ${PID} || :
  fi
}
trap finish EXIT

function e2e-test-setup() {
  echo "- Initializing namespaces"
  kubectl apply -f "${test_dir}/namespaces.yaml"
}

function e2e-test-teardown() {
  echo "- Cleaning up resources"
  kubectl delete --ignore-not-found -f "${test_dir}/namespaces.yaml" --wait
  kubectl delete --ignore-not-found -f "${test_dir}/syncconfig.yaml" --wait
}

function start-espejo() {
  bin/espejo -v &
  PID=$!
  echo "- Started espejo with PID=${PID}"
}
