#!/usr/bin/env bash

set -o errexit
set -o pipefail

# The header file to insert into generated files.
BOILERPLATE="/hack/boilerplate.go.txt"

# The root package under which to search for files which request code to be generated.
PACKAGE_ROOT=""

# The script will only display the commands to be executed without actually executing them.
DRY_RUN=""

# The name of the generated clientset package.
CLIENTSET_NAME="clientset"

# The version of the generated clientset package.
VERSIONED_NAME="versioned"

# Determines how much log output these generator generate.
KUBE_VERBOSE="${KUBE_VERBOSE:-0}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    "--boilerplate")
      BOILERPLATE="$2"
      shift 2
      ;;
    "--pkg-root")
      PACKAGE_ROOT="$2"
      shift 2
      ;;
    "--dry-run")
      DRY_RUN="echo "
      shift 1
      ;;
    "--clientset-name")
      CLIENTSET_NAME="$2"
      shift 2
      ;;
    "--versioned-name")
      VERSIONED_NAME="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

[[ -n "${KUBE_DEBUG:-}" ]] && set -o xtrace

# args: [pkg-root] [regexp]
function kube::codegen::internal::grep_find() {
  find "$(go env GOPATH)/src/$1" -name "*.go" -print0 | xargs -0 grep "$2" | cut -d: -f1
}

# args [version]
function kube::codegen::internal::is_version() {
  echo "$1" | grep -E -q '^v[0-9]+((alpha|beta)[0-9]+)?$'
}

# args [array]
function kube::codegen::internal::not_empty() {
  local input_array="$1"
  [[ -n "$1" && "${#input_array[@]}" != 0 ]]
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::find_pkg() {
  local pkgs=()
  for source in $(kube::codegen::internal::grep_find "$1" "$2"); do
    pkgs+=("$(cd "$(dirname "$source")" && GO111MODULE=on go list -find .)")
  done

  echo "${pkgs[@]}" | xargs -n1 | sort -u
}

# args: [pkg-root]
function kube::codegen::internal::download_mod() {
    cd "$(go env GOPATH)/src/$1" && go mod download
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::remove_match() {
  for source in $(kube::codegen::internal::grep_find "$1" "$2"); do
    rm -rf "$source"
  done
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::apis_base() {
  local gen_pkgs; local apis_pkg; local version

  gen_pkgs=$(kube::codegen::internal::find_pkg "$1" "$2")
  for gen_pkg in $gen_pkgs; do
    version=$(basename "${gen_pkg}")
    if kube::codegen::internal::is_version "${version}"; then
      apis_pkg=$(dirname "$(dirname "${gen_pkg}")")
    fi
  done

  echo "$apis_pkg"
}

# args: [pkg-root] [regexp]
function kube::codegen:internal::inputs() {
    local gen_pkgs

    gen_pkgs=$(kube::codegen::internal::find_pkg "$1" "$2")
    if kube::codegen::internal::not_empty "${gen_pkgs[@]}"; then
      local inputs=()
      for gen_pkg in $gen_pkgs; do
        inputs+=("--input-dirs=${gen_pkg}")
      done
      echo "${inputs[*]}"
    fi
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::gv_inputs() {
  local gen_pkgs

  gen_pkgs=$(kube::codegen::internal::find_pkg "$1" "$2")
  if kube::codegen::internal::not_empty "${gen_pkgs[@]}"; then
    local gv_inputs=()
    for gen_pkg in $gen_pkgs; do
      local group; local version
      version=$(basename "${gen_pkg}")
      if kube::codegen::internal::is_version "${version}"; then
        group=$(basename "$(dirname "${gen_pkg}")")
        gv_inputs+=("--input=${group}/${version}")
      fi
    done
    echo "${gv_inputs[*]}"
  fi
}

# args: [gen] [file_base] [pkg-root] [regexp]
function kube::codegen::internal::gen_helper() {
  local gen_inputs

  kube::codegen::internal::remove_match "$3" "^// Code generated by $1. DO NOT EDIT.$"
  gen_inputs=$(kube::codegen:internal::inputs "$3" "$4")
  if kube::codegen::internal::not_empty "${gen_inputs[@]}"; then
    "${DRY_RUN}$1" -v "$KUBE_VERBOSE" \
      --go-header-file="$BOILERPLATE" \
      --output-file-base="$2" \
      --output-base="$(go env GOPATH)/src" \
      "${gen_inputs[*]}"
  fi
}

# args: [pkg-root]
function kube::codegen::helpers() {
  kube::codegen::internal::gen_helper "deepcopy-gen" "zz_generated.deepcopy" "$1" "+k8s:deepcopy-gen="
  kube::codegen::internal::gen_helper "register-gen" "zz_generated.register" "$1" "+k8s:deepcopy-gen="
  kube::codegen::internal::gen_helper "defaulter-gen" "zz_generated.defaults" "$1" "+k8s:defaulter-gen="
  kube::codegen::internal::gen_helper "conversion-gen" "zz_generated.conversion" "$1" "+k8s:conversion-gen="
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::gen_applyconfiguration() {
  local gen_inputs

  gen_inputs=$(kube::codegen:internal::inputs "$1" "$2")
  if kube::codegen::internal::not_empty "${gen_inputs[@]}"; then
    kube::codegen::internal::remove_match "$3" "^// Code generated by applyconfiguration-gen. DO NOT EDIT.$"
    "${DRY_RUN}applyconfiguration-gen" -v "$KUBE_VERBOSE" \
      --go-header-file="$BOILERPLATE" \
      --output-base="$(go env GOPATH)/src" \
      --output-package="$1/applyconfiguration" \
      "${gen_inputs[@]}"
  fi
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::gen_client() {
  local gv_inputs

  gv_inputs=$(kube::codegen::internal::gv_inputs "$1" "$2")
  if kube::codegen::internal::not_empty "${gv_inputs[@]}"; then
    kube::codegen::internal::remove_match "$3" "^// Code generated by client-gen. DO NOT EDIT.$"
    "${DRY_RUN}client-gen" -v "$KUBE_VERBOSE" \
      --go-header-file="$BOILERPLATE" \
      --clientset-name="$VERSIONED_NAME" \
      --input-base="$(kube::codegen::internal::apis_base "$1" "$2")" \
      --output-base="$(go env GOPATH)/src" \
      --output-package="$1/${CLIENTSET_NAME}" \
      --apply-configuration-package="$1/applyconfiguration" \
      "${gv_inputs[*]}"
  fi
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::gen_lister() {
  local gv_inputs; local gen_inputs

  gv_inputs=$(kube::codegen::internal::gv_inputs "$1" "$2")
  if kube::codegen::internal::not_empty "${gv_inputs[@]}"; then
    kube::codegen::internal::remove_match "$3" "^// Code generated by lister-gen. DO NOT EDIT.$"
    gen_inputs=$(kube::codegen:internal::inputs "$1" "$2")
    "${DRY_RUN}lister-gen" -v "$KUBE_VERBOSE" \
      --go-header-file="$BOILERPLATE" \
      --output-base="$(go env GOPATH)/src" \
      --output-package="$1/listers" \
      "${gen_inputs[*]}"
  fi
}

# args: [pkg-root] [regexp]
function kube::codegen::internal::gen_informer() {
  local gen_inputs; local gen_inputs

  gv_inputs=$(kube::codegen::internal::gv_inputs "$1" "$2")
  if kube::codegen::internal::not_empty "${gv_inputs[@]}"; then
    kube::codegen::internal::remove_match "$3" "^// Code generated by informer-gen. DO NOT EDIT.$"
    gen_inputs=$(kube::codegen:internal::inputs "$1" "$2")
    "${DRY_RUN}informer-gen" -v "$KUBE_VERBOSE" \
      --go-header-file="$BOILERPLATE" \
      --output-base="$(go env GOPATH)/src" \
      --output-package="$1/informers" \
      --versioned-clientset-package="$1/${CLIENTSET_NAME}/${VERSIONED_NAME}" \
      --listers-package="$1/listers" \
      "${gen_inputs[*]}"
  fi
}

# args: [pkg-root]
function kube::codegen::client() {
  kube::codegen::internal::gen_applyconfiguration "$1" "+genclient"
  kube::codegen::internal::gen_client "$1" "+genclient"
  kube::codegen::internal::gen_lister "$1" "+genclient"
  kube::codegen::internal::gen_informer "$1" "+genclient"
}

# args: [pkg-root]
function kube::codegen::all() {
  kube::codegen::internal::download_mod "$1"

  kube::codegen::helpers "$1"
  kube::codegen::client "$1"
}

kube::codegen::all "$PACKAGE_ROOT"
