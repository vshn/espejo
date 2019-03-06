# [Espejo](https://es.wikipedia.org/wiki/Espejo) (object-syncer)

The espejo tool (which means 'mirror' in Spanish) syncs objects from a SyncConfig CRD to multiple namespaces. The idea is to replace OpenShift's [project templates](https://docs.openshift.com/container-platform/3.9/admin_guide/managing_projects.html#modifying-the-template-for-new-projects) with a more flexible and robust solution.

## CustomResourceDefinitions
The operator introduces a CRD called `SyncConfig` to configure the objects which should be synced:
```yaml
apiVersion: sync.appuio.ch/v1alpha1
kind: SyncConfig
metadata:
  name: example
spec:
  forceRecreate: false
  namespaceSelector:
    labelSelector:
      matchExpressions:
      - key: customer
        operator: Exists
      matchLabels:
        some-label: value
    matchNames:
    - myproject
  items:
  - apiVersion: v1
    kind: Service
    metadata:
      name: glusterfs-cluster
    spec:
      type: ClusterIP
      clusterIP: None
      ports:
      - port: 49152
        targetPort: 49152
        protocol: TCP
  - apiVersion: v1
    kind: Endpoints
    metadata:
      name: glusterfs-cluster
    subsets:
    - addresses:
      - ip: 172.28.54.121
      - ip: 172.28.54.122
      - ip: 172.28.54.123
      ports:
      - port: 49152
        protocol: TCP
  - apiVersion: extensions/v1beta1
    kind: NetworkPolicy
    metadata:
      name: allow-from-same-namespace
    spec:
      ingress:
      - from:
        - podSelector: {}
      podSelector: {}
      policyTypes:
      - Ingress
  deleteItems:
  - apiVersion: v1
    kind: ConfigMap
    name: some-name
```
This `SyncConfig` will create a `Service`, `Endpoints` and `NetworkPolicy` object in all namepsaces which mach the [label selector](https://v1-9.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#labelselector-v1-meta) OR one of the name selectors.
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

The parameter replacement is implemented using the [OpenShift templates pkg](https://github.com/openshift/origin/tree/master/pkg/template/templateprocessing) and therefore follows the same [rules](https://docs.openshift.com/container-platform/3.9/dev_guide/templates.html#writing-parameters).

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
