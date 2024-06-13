//go:build generate

package main

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=${CRD_ROOT_DIR}/v1/base crd:crdVersions=v1,deprecatedV1beta1CompatibilityPreserveUnknownFields=false
