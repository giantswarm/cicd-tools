# repo: giantswarm/kubectl-gs
ARG KUBECTL_GS_VERSION=2.31.1
# repo: giantswarm/opsctl
ARG OPSCTL_VERSION=3.3.0
FROM quay.io/giantswarm/kubectl-gs:${KUBECTL_GS_VERSION} AS kubectl-gs
FROM quay.io/giantswarm/opsctl:${OPSCTL_VERSION} AS opsctl

FROM ubuntu:latest

ENV TZ=Etc/UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN apt update &&\
    apt install -y make git curl golang-go jq

# kubectl
ARG KUBECTL_VERSION=
RUN KUBECTL_VERSION=${KUBECTL_VERSION:=$(curl -L -s https://dl.k8s.io/release/stable.txt)} && \
    curl -s -LO "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" && \
    curl -s -LO "https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

## Install helm
ARG HELM_VERSION=
RUN HELM_VERSION=${HELM_VERSION:=$(curl -s "https://api.github.com/repos/helm/helm/releases/latest" | jq -r '.tag_name | ltrimstr("v")')} && \
    curl -s -L https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar xvz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm

# kubectl-gs
COPY --from=kubectl-gs /usr/bin/kubectl-gs /usr/bin/kubectl-gs

# opsctl
COPY --from=opsctl /usr/local/bin/opsctl /usr/local/bin/opsctl

# vcluster
ARG VCLUSTER_VERSION=
RUN VCLUSTER_VERSION=${VCLUSTER_VERSION:=$(curl -s https://api.github.com/repos/loft-sh/vcluster/releases/latest | jq -r '.tag_name')} && \
    curl -s -LO "https://github.com/loft-sh/vcluster/releases/download/${VCLUSTER_VERSION}/vcluster-linux-amd64" && \
    chmod +x vcluster-linux-amd64 && \
    mv vcluster-linux-amd64 /usr/local/bin/vcluster

# GH cli
ARG GH_VERSION=
RUN GH_VERSION=${GH_VERSION:=$(curl -s "https://api.github.com/repos/cli/cli/releases/latest" | jq -r '.tag_name | ltrimstr("v")')} &&\
    curl -fsL -o "/tmp/gh.deb" "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.deb" && \
    dpkg -i /tmp/gh.deb

RUN mkdir /scripts

COPY scripts /scripts
