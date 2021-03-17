// +build tools

// Package tools contains any runtime Go dependencies as imports.
// Go modules will be forced to download and install them.
package tools

import (
	// used to generate DeepCopy methods
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
	// used to run e2e tests
	_ "sigs.k8s.io/kustomize/kustomize/v3"
)
