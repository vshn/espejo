#!/bin/bash

function e2e-test-setup() {
  echo "Initializing namespaces"
  kubectl apply -f ${test_dir}/namespaces.yaml
}

function e2e-test-teardown() {
  echo "Cleaning up resources"
  kubectl delete -f ${test_dir}/namespaces.yaml
  kubectl delete -f ${test_dir}/syncconfig.yaml
}
