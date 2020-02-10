# [Espejo](https://es.wikipedia.org/wiki/Espejo) (object-syncer)

The espejo tool (which means 'mirror' in Spanish) syncs objects from a SyncConfig CRD to multiple namespaces. The idea is to replace OpenShift's [project templates](https://docs.openshift.com/container-platform/3.11/admin_guide/managing_projects.html#modifying-the-template-for-new-projects) with a more flexible and robust solution.

## CustomResourceDefinitions
The operator introduces a CRD called `SyncConfig` to configure the objects which should be synced (see [example](deploy/crds/sync_v1alpha1_syncconfig_cr.yaml)).
[This `SyncConfig`](deploy/crds/sync_v1alpha1_syncconfig_cr.yaml) will create a `Service`, `Endpoints`, `NetworkPolicy` and `HorizontalPodAutoscaler` objects in all namepsaces which mach the [label selector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.15/#labelselector-v1-meta) OR one of the name selectors.
To ensure objects are deleted, set the `prune` parameter to `true` (default is `false`)

### Parameters
The following parameters can be used inside object definitions:

| Parameter Name               | Description                  |
|------------------------------|------------------------------|
| `${PROJECT_NAME}`            | Name of the target namespace |
| `${PROJECT_DISPLAYNAME}`     | Not implemented              |
| `${PROJECT_DESCRIPTION}`     | Not implemented              |
| `${PROJECT_ADMIN_USER}`      | Not implemented              |
| `${PROJECT_REQUESTING_USER}` | Not implemented              |

The parameter replacement is implemented using the [OpenShift templates pkg](https://github.com/openshift/origin/tree/release-3.11/pkg/template/templateprocessing) and therefore follows the same [rules](https://docs.openshift.com/container-platform/3.11/dev_guide/templates.html#writing-parameters).

## Installation


## Development
The tool is implemented with the [Operator SDK](https://github.com/operator-framework/operator-sdk).

1. [Install the Operator SDK](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#install-the-operator-sdk-cli)
2. Install dependencies
```bash
dep ensure
```
3. Run the operator
```bash
cd cmd/manager
go run main.go
```
