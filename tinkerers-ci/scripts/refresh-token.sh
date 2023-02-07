#!/usr/bin/env bash

#-------------------------
# Environment variables that need to be set
#-------------------------

# github app id
#app_id=284804

# github app installation id
#installation_id=33432247

# pem file generated for github-app
# app_private_key=""

#-------------------------
# Generate JWT using pem 
#-------------------------

header='{
    "alg": "RS256",
    "typ": "JWT"
}'

build_payload() {
        # shellcheck disable=SC2154
        jq -n -j -c \
                --arg iat_str "$(date +%s)" \
                --arg app_id "${app_id}" \
        '
        ($iat_str | tonumber) as $iat
        | .iat = $iat
        | .exp = ($iat + 300)
        | .iss = ($app_id | tonumber)
        '
}

b64enc() { openssl enc -base64 -A | tr '+/' '-_' | tr -d '='; }
rs256_sign() { openssl dgst -binary -sha256 -sign <(printf '%s\n' "$1"); }

algo=${1:-RS256}; 
algo=${algo^^} # Convert to uppercase 
payload=$(build_payload) || return
signed_content="$(jq -c -j -n "$header" | b64enc).$(jq -c -j -n "$payload" | b64enc)"
# shellcheck disable=SC2154
sig=$(printf %s "$signed_content" | rs256_sign "$app_private_key" | b64enc)
generated_jwt=$(printf '%s.%s\n' "${signed_content}" "${sig}")

#-------------------------
## Generate token using jwt 
#-------------------------

GITHUB_TOKEN=$(curl \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${generated_jwt}" \
  https://api.github.com/app/installations/${installation_id}/access_tokens \
  2>/dev/null | \
  jq -e '.token' | sed 's/\"//g' )	 


#-------------------------
# Update stored secret with new token 
#-------------------------

set +e
kubectl delete secret -n tekton-pipelines tinkerers-ci-github-token 
set -e

kubectl -n tekton-pipelines create secret generic tinkerers-ci-github-token \
  --from-literal=GITHUB_TOKEN="${GITHUB_TOKEN}"


#-------------------------
# basic-access-auth secret stored in flux-system namespace
#-------------------------

set +e
kubectl delete secret -n flux-system basic-access-auth
set -e

kubectl -n flux-system create secret generic basic-access-auth \
--from-literal=password="${GITHUB_TOKEN}" \
--from-literal=username=x-access-token

