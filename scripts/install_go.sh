#!/bin/bash

set -e
set -u
set -o pipefail

function main() {
  if [[ "${CF_STACK:-}" != "cflinuxfs4" ]]; then
      echo "       **ERROR** Unsupported stack"
      echo "                 See https://docs.cloudfoundry.org/devguide/deploy-apps/stacks.html for more info"
      exit 1
  fi

  local version expected_sha dir
  version="1.20.2"
  expected_sha="3f0935974b213d5b53b72db935b1de1582a7125a4510f616820cc2fc51be2eda"
  dir="/tmp/go${version}"

  mkdir -p "${dir}"

  if [[ ! -f "${dir}/go/bin/go" ]]; then
    local url

    url="https://buildpacks.cloudfoundry.org/dependencies/go/go_${version}_linux_x64_${CF_STACK}_${expected_sha:0:8}.tgz"

    echo "-----> Downloading Go ${version}"
    curl "${url}" \
      --silent \
      --location \
      --retry 15 \
      --retry-delay 3 \
      --output "/tmp/go.tgz"

    local sha
    sha="$(shasum -a 256 /tmp/go.tgz | cut -d ' ' -f 1)"

    if [[ "${sha}" != "${expected_sha}" ]]; then
      echo "       **ERROR** Go SHA256 mismatch: got ${sha}, expected ${expected_sha}"
      exit 1
    fi

    tar xzf "/tmp/go.tgz" -C "${dir}"
    rm "/tmp/go.tgz"
  fi

  if [[ ! -f "${dir}/bin/go" ]]; then
    echo "       **ERROR** Could not download Go"
    exit 1
  fi

  GOROOT_TEMP="${dir}"
  export GOROOT_TEMP
}

main "${@:-}"
