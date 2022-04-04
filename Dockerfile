# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# set auth credentials via .netrc for private cesapp repository
COPY .netrc /root/.netrc

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN GOPRIVATE=github.com/cloudogu/cesapp/v4 go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -a -o k8s-dogu-operator main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-dogu-operator" \
      VERSION="0.2.0"

WORKDIR /
COPY --from=builder /workspace/k8s-dogu-operator .
# the linter has a problem with the valid colon-syntax
# dockerfile_lint - ignore
USER 65532:65532

ENTRYPOINT ["/k8s-dogu-operator"]
