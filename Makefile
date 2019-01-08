SHELL=/bin/bash -o pipefail

GO ?= go
DOCKER ?= docker

COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),${COMMIT_NO}-dirty,${COMMIT_NO})
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")

IMAGE_TRACERUNNER_BRANCH := quay.io/fntlnz/kubectl-trace-bpftrace:$(GIT_BRANCH_CLEAN)
IMAGE_TRACERUNNER_COMMIT := quay.io/fntlnz/kubectl-trace-bpftrace:$(GIT_COMMIT)
IMAGE_TRACERUNNER_LATEST := quay.io/fntlnz/kubectl-trace-bpftrace:latest

BPFTRACESHA ?= 2ae2a53f62622631a304def6c193680e603994e3
IMAGE_BPFTRACE_BASE := quay.io/fntlnz/kubectl-trace-bpftrace-base:$(BPFTRACESHA)

IMAGE_BUILD_FLAGS ?= "--no-cache"

LDFLAGS := -ldflags '-X github.com/iovisor/kubectl-trace/pkg/version.buildTime=$(shell date +%s) -X github.com/iovisor/kubectl-trace/pkg/version.gitCommit=${GIT_COMMIT}'
TESTPACKAGES := $(shell go list ./... | grep -v github.com/iovisor/kubectl-trace/integration)

kubectl_trace ?= _output/bin/kubectl-trace
trace_runner ?= _output/bin/trace-runner

.PHONY: build
build: clean ${kubectl_trace}

${kubectl_trace}:
	CGO_ENABLED=1 $(GO) build ${LDFLAGS} -o $@ ./cmd/kubectl-trace

${trace_runner}:
	CGO_ENABLED=1 $(GO) build ${LDFLAGS} -o $@ ./cmd/trace-runner

.PHONY: clean
clean:
	rm -Rf _output

.PHONY: image/build
image/build:
	$(DOCKER) build --build-arg bpftracesha=$(BPFTRACESHA) $(IMAGE_BUILD_FLAGS) -t $(IMAGE_TRACERUNNER_BRANCH) -f Dockerfile.tracerunner .
	$(DOCKER) tag $(IMAGE_TRACERUNNER_BRANCH) $(IMAGE_TRACERUNNER_COMMIT)

.PHONY: image/push
image/push:
	$(DOCKER) push $(IMAGE_TRACERUNNER_BRANCH)
	$(DOCKER) push $(IMAGE_TRACERUNNER_COMMIT)

.PHONY: image/latest
image/latest:
	$(DOCKER) tag $(IMAGE_TRACERUNNER_COMMIT) $(IMAGE_TRACERUNNER_LATEST)
	$(DOCKER) push $(IMAGE_TRACERUNNER_LATEST)

.PHONY: test
test:
	$(GO) test -v -race $(TESTPACKAGES)

.PHONY: integration
integration:
	TEST_KUBECTLTRACE_BINARY=$(shell pwd)/$(kubectl_trace) $(GO) test -v ./integration/...

.PHONY: bpftraceimage/build
bpftraceimage/build:
	$(DOCKER) build --build-arg bpftracesha=$(BPFTRACESHA) $(IMAGE_BUILD_FLAGS) -t $(IMAGE_BPFTRACE_BASE) -f Dockerfile.bpftracebase .

.PHONY: bpftraceimage/push
bpftraceimage/push:
	$(DOCKER) push $(IMAGE_BPFTRACE_BASE)
