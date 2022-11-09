# Build the manager binary
FROM golang:1.18 as builder

ENV GOPRIVATE=github.com/cloudogu/cesapp/v5

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY retry/ retry/

# Copy .git files as the build process builds the current commit id into the binary via ldflags.
# We removed this entry as changes in the repository makes all cached layers invalid leading to rebuilding all layers.
# TODO resolve COMMIT_ID
#COPY .git .git

# Copy build files
COPY build build
COPY Makefile Makefile

RUN mkdir /tmp/dogu-registry-cache

# Build
RUN go mod vendor
RUN make compile-generic

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-dogu-operator" \
      VERSION="0.13.0"

WORKDIR /
COPY --from=builder /workspace/target/k8s-dogu-operator .
COPY --from=builder --chown=65532:65532 /tmp/dogu-registry-cache /tmp/dogu-registry-cache

# the linter has a problem with the valid colon-syntax
# dockerfile_lint - ignore
USER 65532:65532

ENTRYPOINT ["/k8s-dogu-operator"]
