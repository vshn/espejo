//+build generate

package main

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:trivialVersions=true rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=${CRD_ROOT_DIR}/v1beta1/base crd:crdVersions=v1beta1
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:trivialVersions=true rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=${CRD_ROOT_DIR}/v1/base      crd:crdVersions=v1

// Run this file itself
//go:generate go run generate.go v1/base/sync.appuio.ch_syncconfigs.yaml v1
//go:generate go run generate.go v1beta1/base/sync.appuio.ch_syncconfigs.yaml v1beta1

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// controller-gen 0.3 creates CRDs with apiextensions.k8s.io/v1beta1, but some generated properties aren't valid for that version
// in K8s 1.18+. We would have to switch to apiextensions.k8s.io/v1, but that would make the CRD incompatible with OpenShift 3.11.
// So we have to patch the CRD in post-generation.
// See https://github.com/kubernetes/kubernetes/issues/91395
func main() {
	workdir, _ := os.Getwd()
	log.Println("Running post-generate in " + workdir)
	file := os.Args[1]
	fileName := os.Getenv("CRD_ROOT_DIR") + "/" + file
	crdVersion := os.Args[2]
	patchFile(fileName, crdVersion)
}

func patchFile(fileName, version string) {
	log.Println(fmt.Sprintf("Reading file %s", fileName))
	lines, err := readLines(fileName)
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}
	var result []string
	switch version {
	case "v1":
		result = patchV1(lines, result)
	case "v1beta1":
		result = patchV1beta1(lines, result)
	}

	log.Println(fmt.Sprintf("Writing new file to %s", fileName))
	if err := writeLines(result, fileName); err != nil {
		log.Fatalf("writeLines: %s", err)
	}
}

func patchV1(lines []string, result []string) []string {
	for i, line := range lines {
		switch line {
		case "                  type: object":
			result = append(result, line)
			affectsSyncItems := strings.Contains(lines[i-2], "description: SyncItems")

			if affectsSyncItems {
				hasEmbeddedResource := strings.Contains(lines[i+2], "x-kubernetes-embedded-resource")
				if hasEmbeddedResource {
					result = append(result, "                  x-kubernetes-embedded-resource: true")
					log.Println(fmt.Sprintf("Added  'x-kubernetes-embedded-resource' after line %d", i))
				}
				hasPreserveUnknownFields := strings.Contains(lines[i+3], "x-kubernetes-preserve-unknown-fields")
				if hasPreserveUnknownFields {
					result = append(result, "                  x-kubernetes-preserve-unknown-fields: true")
					log.Println(fmt.Sprintf("Added  'x-kubernetes-preserve-unknown-fields' after line %d", i))
				}
			}
		case "                x-kubernetes-embedded-resource: true":
			log.Println(fmt.Sprintf("Removed 'x-kubernetes-embedded-resource' in line %d", i))
		case "                x-kubernetes-preserve-unknown-fields: true":
			log.Println(fmt.Sprintf("Removed 'x-kubernetes-preserve-unknown-fields' in line %d", i))
		default:
			result = append(result, line)
		}
	}
	return result
}

func patchV1beta1(lines []string, result []string) []string {
	for i, line := range lines {
		switch line {
		case "              x-kubernetes-embedded-resource: true":
			log.Println(fmt.Sprintf("Removed 'x-kubernetes-embedded-resource' in line %d", i))
		case "              x-kubernetes-preserve-unknown-fields: true":
			log.Println(fmt.Sprintf("Removed 'x-kubernetes-preserve-unknown-fields' in line %d", i))
		default:
			result = append(result, line)
		}
	}
	return result
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}
