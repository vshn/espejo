#!/bin/bash

errcho() {
	>&2 echo "${@}"
}

if [ -z "${E2E_IMAGE}" ]; then
	errcho "The environment variable 'E2E_IMAGE' is undefined or empty."
	exit 1
fi

setup() {
	debug "-- $BATS_TEST_DESCRIPTION"
	debug "-- $(date)"
	debug ""
	debug ""
}

setup_file() {
	reset_debug
}

teardown() {
	cp -r /tmp/detik debug || true
}

kustomize() {
	go run sigs.k8s.io/kustomize/kustomize/v3 "${@}"
}

replace_in_file() {
	VAR_NAME=${1}
	VAR_VALUE=${2}
	FILE=${3}

	sed -i \
		-e "s|\$${VAR_NAME}|${VAR_VALUE}|" \
		"${FILE}"
}

prepare() {
	DEFINITION_DIR=${1}
	mkdir -p "debug/${DEFINITION_DIR}"
	kustomize build "${DEFINITION_DIR}" -o "debug/${DEFINITION_DIR}/main.yml"

	replace_in_file E2E_IMAGE "'${E2E_IMAGE}'" "debug/${DEFINITION_DIR}/main.yml"
	replace_in_file ID "$(id -u)" "debug/${DEFINITION_DIR}/main.yml"
	replace_in_file ENABLE_LEADER_ELECTION "'${ENABLE_LEADER_ELECTION}'" "debug/${DEFINITION_DIR}/main.yml"
}

apply() {
	prepare "${@}"
	kubectl apply -f "debug/${1}/main.yml" --validate=false
}

given_running_operator() {
	apply definitions/operator
}

wait_until() {
	object=${1}
	condition=${2}
	kubectl -n "${DETIK_CLIENT_NAMESPACE}" wait --timeout 1m --for "condition=${condition}" "${object}"
}
