#!/usr/bin/env bash

k8s_version=1.23.5

set -x 

YQ="yq"
KUBECTL="kubectl"
VCLUSTER=vcluster

KUBECONFIG_FOLDER=./


"${KUBECTL}" create namespace "${INSTALLATION}-vcluster" --dry-run=client -o yaml | "${KUBECTL}" apply -f -

cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: vcluster-${INSTALLATION}-role-binding
  namespace: ${INSTALLATION}-vcluster
subjects:
- kind: ServiceAccount
  name: vc-${INSTALLATION}-vcluster 
roleRef:
  kind: ClusterRole 
  name: privileged-psp-user 
  apiGroup: rbac.authorization.k8s.io
EOF

# Grab the basedomain from the nginx ingress controller app chart values file

baseDomain=$("${KUBECTL}" get cm -n giantswarm nginx-ingress-controller-app-chart-values --output="jsonpath={.data.values}" | ${YQ} '.baseDomain')

cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: HTTPS
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  name: vcluster-ingress
  namespace: ${INSTALLATION}-vcluster
spec:
  ingressClassName: nginx 
  rules:
  - host: ${INSTALLATION}-vcluster.${baseDomain}
    http:
      paths:
      - backend:
          service:
            name: ${INSTALLATION}-vcluster
            port: 
              number: 443
        path: /
        pathType: ImplementationSpecific
EOF


cat <<EOF > values.yaml
vcluster:
    image: rancher/k3s:v${k8s_version}-k3s1    
syncer:
    extraArgs:
    - --tls-san="${INSTALLATION}-vcluster.${baseDomain}"
EOF


echo "---> Create and waiting for vcluster to be ready"
while true ; do

    "${VCLUSTER}" create "${INSTALLATION}-vcluster" -n "${INSTALLATION}-vcluster" --upgrade --connect=false -f ./values.yaml

    set -x
    sleep 4
    { set +x; } 2>/dev/null

    ${VCLUSTER} connect "${INSTALLATION}-vcluster" \
        -n "${INSTALLATION}-vcluster" \
        --update-current=false \
        --service-account admin \
        --cluster-role cluster-admin \
        --insecure \
        --server="https://${INSTALLATION}-vcluster.${baseDomain}"

    cp ./kubeconfig.yaml "${KUBECONFIG_FOLDER}/${BOOTSTRAP_CLUSTER}.kubeconfig"

    set +e
    set -x
    "${KUBECTL}" --kubeconfig=./kubeconfig.yaml get nodes
    res=$?
    { set +x; } 2>/dev/null
    set -e
    if [[ ${res} -eq 0 ]] ; then
        break
    fi
done


readonly kubeconfig_path="${KUBECONFIG_FOLDER}/${INSTALLATION}.kubeconfig"


echo "==> Storing kubeconfig as a secret"

set +e
"${KUBECTL}" delete secret \
    "${INSTALLATION}-kubeconfig" \
    -n tekton-ci
set -e

"${KUBECTL}" create secret generic \
    "${INSTALLATION}-kubeconfig" \
    -n tekton-ci \
    --from-file=kubeconfig="${kubeconfig_path}"