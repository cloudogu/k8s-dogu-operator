ARG HELMFILE_VERSION=v1.8.1
FROM ghcr.io/helmfile/helmfile:${HELMFILE_VERSION} AS helmfile

FROM golang:1.27.1 AS builder

WORKDIR /workspace

ARG GOBIN=/workspace
ARG HELM_PLUGINS=/helm_plugins

# openDesk needs Helm >= v3.17.3 and < v4.x1 but not v3.18.02 or v3.20.13
ARG HELM_VERSION=v3.22.0
ARG HELM_DIFF_VERSION=v3.15.15

RUN go install "helm.sh/helm/v3/cmd/helm@${HELM_VERSION}"
RUN mkdir "${HELM_PLUGINS}" && \
    /workspace/helm plugin install https://github.com/databus23/helm-diff --version "${HELM_DIFF_VERSION}"

FROM scratch

COPY --from=helmfile /usr/local/bin/helmfile /
COPY --from=builder /workspace /
COPY --from=builder /helm_plugins/helm-diff/bin /helm_plugins/bin
COPY --from=builder /helm_plugins/helm-diff/plugin.yaml /helm_plugins/