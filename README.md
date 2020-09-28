# [Espejo](https://es.wikipedia.org/wiki/Espejo) (object-syncer)

[![Build](https://img.shields.io/github/workflow/status/vshn/espejo/Build)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/espejo)
![Kubernetes version](https://img.shields.io/badge/k8s-v1.18-blue)
[![Version](https://img.shields.io/github/v/release/vshn/espejo)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/espejo/total)][releases]
[![Docker image](https://img.shields.io/docker/pulls/vshn/espejo)][dockerhub]
[![License](https://img.shields.io/github/license/vshn/espejo)][license]

The espejo tool (which means 'mirror' in Spanish) syncs objects from a SyncConfig CRD to multiple namespaces. The idea is to replace OpenShift's [project templates](https://docs.openshift.com/container-platform/3.11/admin_guide/managing_projects.html#modifying-the-template-for-new-projects) with a more flexible and robust solution.

## CustomResourceDefinitions

The operator introduces a CRD called `SyncConfig` to configure the objects which should be synced.
[This `SyncConfig`](example/syncconfig.yaml) will create a `Service`, `Endpoints` and `NetworkPolicy` object in all namepsaces which mach the [label selector](https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#labelselector-v1-meta) OR one of the name selectors.
To ensure objects are deleted, set the `prune` parameter to `true` (default is `false`)

### Parameters

Strings within object definitions can be replaced with dynamic values with parameters. The following parameters can be used:

| Parameter Name               | Description                  |
|------------------------------|------------------------------|
| `${PROJECT_NAME}`            | Name of the target namespace |

## Development

The Operator is implemented with the [Operator SDK](https://github.com/operator-framework/operator-sdk) ([Installation](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/)).

### Build

`make build` places the binary in `bin/espejo`. Go is required.

### Integration tests

`make integration_test` runs unit test cases with a K8s test environment.

### End-to-End tests

`make e2e_test` will run [Kubernetes in Docker](https://kind.sigs.k8s.io/) to simulate a real-world K8s cluster so we can install and run the operator on it. Docker and Kubectl are required.
