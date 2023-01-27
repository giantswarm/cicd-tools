#!/usr/bin/env bash

GITHUB_TOKEN=$( /scripts/generate_token.sh )

set +e
kubectl delete secret -n -ci generic tinkerers-ci-github-token 
set -e

kubectl create secret -n -ci generic tinkerers-ci-github-token \
  --from-literal=GITHUB_TOKEN="${GITHUB_TOKEN}"