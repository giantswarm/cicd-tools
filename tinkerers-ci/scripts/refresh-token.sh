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
payload_template='{}'

build_payload() {
        # shellcheck disable=SC2154
        jq -c \
                --arg iat_str "$(date +%s)" \
                --arg app_id "${app_id}" \
        '
        ($iat_str | tonumber) as $iat
        | .iat = $iat
        | .exp = ($iat + 300)
        | .iss = ($app_id | tonumber)
        ' <<< "${payload_template}" | tr -d '\n'
}

b64enc() { openssl enc -base64 -A | tr '+/' '-_' | tr -d '='; }
json() { jq -c . | LC_CTYPE=C tr -d '\n'; }
rs256_sign() { openssl dgst -binary -sha256 -sign <(printf '%s\n' "$1"); }

algo=${1:-RS256}; algo=${algo^^}
payload=$(build_payload) || return
signed_content="$(json <<<"$header" | b64enc).$(json <<<"$payload" | b64enc)"
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
kubectl delete secret -n -ci generic tinkerers-ci-github-token 
set -e

kubectl create secret -n -ci generic tinkerers-ci-github-token \
  --from-literal=GITHUB_TOKEN="${GITHUB_TOKEN}"