# Build the manager binary
FROM golang:1.18 as builder

ENV GOPRIVATE=github.com/cloudogu/cesapp/v5

WORKDIR /workspace

# set auth credentials via .netrc for private cesapp repository
COPY .netrc /root/.netrc

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

# Copy .git files as the build process builds the current commit id into the binary via ldflags
COPY .git .git

# Copy build files
COPY build build
COPY Makefile Makefile

# Build
RUN go mod vendor
RUN make compile-generic

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-dogu-operator" \
      VERSION="0.10.0"

WORKDIR /
COPY --from=builder /workspace/target/k8s-dogu-operator .
# the linter has a problem with the valid colon-syntax
# dockerfile_lint - ignore
USER 65532:65532

ENTRYPOINT ["/k8s-dogu-operator"]
