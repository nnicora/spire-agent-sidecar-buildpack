#!/bin/bash

set -e
set -u
set -o pipefail


function main() {
  local shas expected_sha version dir
   version="1.19"
   shas["cflinuxfs3"]="7e231ea5c68f4be7fea916d27814cc34b95e78c4664c3eb2411e8370f87558bd"

#  if [[ "${CF_STACK:-}" != "cflinuxfs3" || "${CF_STACK:-}" != "cflinuxfs4" ]]; then
#      CF_STACK="cflinuxfs3"
#  fi

  if [[ "${CF_STACK:-}" != "cflinuxfs3" ]]; then
      CF_STACK="cflinuxfs3"
  fi
  expected_sha=shas["${CF_STACK}"]

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