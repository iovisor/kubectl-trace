SHELL=/bin/bash -o pipefail

GO ?= go
DOCKER ?= docker

COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),${COMMIT_NO}-dirty,${COMMIT_NO})
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")

IMAGE_BPFTRACE_BRANCH := quay.io/fntlnz/kubectl-trace-bpftrace:$(GIT_BRANCH_CLEAN)
IMAGE_BPFTRACE_COMMIT := quay.io/fntlnz/kubectl-trace-bpftrace:$(GIT_COMMIT)

IMAGE_BUILD_FLAGS ?= "--no-cache"

LDFLAGS := -ldflags '-X github.com/fntlnz/kubectl-trace/pkg/version.buildTime=$(shell date +%s) -X github.com/fntlnz/kubectl-trace/pkg/version.gitCommit=${GIT_COMMIT}'

kubectl_trace ?= _output/bin/kubectl-trace
trace_runner ?= _output/bin/trace-runner

.PHONY: build
build: clean ${kubectl_trace} ${trace_runner}

${kubectl_trace}:
	CGO_ENABLED=1 $(GO) build ${LDFLAGS} -o $@ ./cmd/kubectl-trace

${trace_runner}:
	CGO_ENABLED=1 $(GO) build ${LDFLAGS} -o $@ ./cmd/trace-runner

.PHONY: clean
clean:
	rm -Rf _output

.PHONY: image/build
image/build:
	$(DOCKER) build $(IMAGE_BUILD_FLAGS) -t $(IMAGE_BPFTRACE_BRANCH) -f Dockerfile.bpftrace .
	$(DOCKER) tag $(IMAGE_BPFTRACE_BRANCH) $(IMAGE_BPFTRACE_COMMIT)

.PHONY: image/push
image/push:
	$(DOCKER) push $(IMAGE_BPFTRACE_BRANCH)
	$(DOCKER) push $(IMAGE_BPFTRACE_COMMIT)
