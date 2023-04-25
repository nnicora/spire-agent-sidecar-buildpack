#!/bin/bash

set -e
set -u
set -o pipefail


function main() {
  if [[ "${CF_STACK:-}" != "cflinuxfs3" || "${CF_STACK:-}" != "cflinuxfs4" ]]; then
      CF_STACK="cflinuxfs4"
  fi

 local expected_sha version dir
 version="1.20.3"
  if [[ "${CF_STACK:-}" == "cflinuxfs3" ]]; then
        expected_sha="02e80e1f944e22bb38ea99337d5a62ab7567b9a5615c19931e3a36749d28c415"
  fi
  if [[ "${CF_STACK:-}" == "cflinuxfs4" ]]; then
        expected_sha="69f652d7f6fdaf9b12e721899ba076e5298cb668623e143bb5b4d83068501aca"
  fi

  if [ -z ${expected_sha+x} ]; then
      echo "  **ERROR** Unsupported stack"
      echo "    See https://docs.cloudfoundry.org/devguide/deploy-apps/stacks.html for more info"
      exit 1
  fi

  echo "Using CF stack ${CF_STACK}"

  dir="/tmp/go${version}"

  mkdir -p "${dir}"

  if [[ ! -f "${dir}/go/bin/go" ]]; then
    local url
    url="https://buildpacks.cloudfoundry.org/dependencies/go/go_${version}_linux_x64_${CF_STACK}_${expected_sha:0:8}.tgz"

    echo "-----> Download Golang Buildpack: ${url}"
    curl "${url}" \
      --silent \
      --location \
      --retry 15 \
      --retry-delay 4 \
      --output "/tmp/go.tgz"

    local sha
    sha="$(shasum -a 256 /tmp/go.tgz | cut -d ' ' -f 1)"

    if [[ "${sha}" != "${expected_sha}" ]]; then
      echo "       **ERROR** Golang Buildpack SHA256 mismatch: got ${sha}, expected ${expected_sha}"
      exit 1
    fi

    tar xzf "/tmp/go.tgz" -C "${dir}"
    rm "/tmp/go.tgz"
  fi

  if [[ ! -f "${dir}/bin/go" ]]; then
    echo "       **ERROR** Could not download go from set URL: ${url}"
    exit 1
  fi

  GoInstallDir="${dir}"
  export GoInstallDir
}

main "${@:-}"