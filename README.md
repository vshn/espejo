# [Espejo](https://es.wikipedia.org/wiki/Espejo) (object-syncer)

[![Build](https://img.shields.io/github/workflow/status/vshn/espejo/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/espejo)
![Kubernetes version](https://img.shields.io/badge/k8s-v1.20-blue)
[![Version](https://img.shields.io/github/v/release/vshn/espejo)][releases]
[![Maintainability](https://img.shields.io/codeclimate/maintainability/vshn/espejo)][codeclimate]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/espejo/total)][releases]
[![Docker image](https://img.shields.io/docker/pulls/vshn/espejo)][dockerhub]
[![License](https://img.shields.io/github/license/vshn/espejo)][license]

The espejo tool (which means 'mirror' in Spanish) syncs objects from a SyncConfig CRD to multiple namespaces. The idea is to replace OpenShift's [project templates](https://docs.openshift.com/container-platform/3.11/admin_guide/managing_projects.html#modifying-the-template-for-new-projects) with a more flexible and robust solution.

## CustomResourceDefinitions

The operator introduces a CRD called `SyncConfig` to configure the objects which should be synced.
[This `SyncConfig`](config/samples/complete-syncconfig.yaml) will create a `Service`, `Endpoints` and `NetworkPolicy` object in all namespaces which mach the [label selector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#labelselector-v1-meta) OR one of the name selectors.
To ensure objects are deleted, set the `prune` parameter to `true` (default is `false`)

### Parameters

Strings within object definitions can be replaced with dynamic values with parameters. The following parameters can be used:

| Parameter Name               | Description                  |
|------------------------------|------------------------------|
| `${PROJECT_NAME}`            | Name of the target namespace |

## Development

The Operator is implemented with the [Operator SDK](https://github.com/operator-framework/operator-sdk) ([Installation](https://sdk.operatorframework.io/docs/installation/)).

### Build

* `make build` creates the `espejo` binary. Go is required.
* `make docker-build` creates the Docker image with `docker.io/vshn/espejo:latest` and `quay.io/vshn/espejo:latest` tags.
* `make test` runs all unit tests.
* `make integration-test` runs the integration tests.

### Run E2E tests

You need `node` and `npm` to run the tests, as it runs with [DETIK][detik].

To run e2e tests, execute:

```bash
make e2e-test
```

[build]: https://github.com/vshn/espejo/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/espejo/releases
[license]: https://github.com/vshn/espejo/blob/master/LICENSE
[dockerhub]: https://hub.docker.com/r/vshn/espejo
[codeclimate]: https://codeclimate.com/github/vshn/espejo

[detik]: https://github.com/bats-core/bats-detik
