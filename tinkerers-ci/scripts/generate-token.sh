#!/usr/bin/env bash

# REF: https://docs.github.com/en/rest/apps/apps#create-an-installation-access-token-for-an-app

# To add more permissions or edit them navigate here.
# https://github.com/organizations/giantswarm/settings/installations/31296181

dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export app_id=284804
#app_private_key env variable needs to be set
generated_jwt=$("${dir}/generate-jwt.sh")
readonly generated_jwt

installation_id=33432247
readonly installation_id

GITHUB_TOKEN=$(curl \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${generated_jwt}" \
  https://api.github.com/app/installations/${installation_id}/access_tokens \
  2>/dev/null | \
  jq -e '.token' | sed 's/\"//g' )	 

echo "${GITHUB_TOKEN}"
