ARG bpftracesha
ARG imagenamebase

FROM ${imagenamebase}:${bpftracesha} as bpftrace
FROM golang:1.11.4-alpine3.8 as gobuilder

RUN apk update
RUN apk add make bash git

ADD . /go/src/github.com/iovisor/kubectl-trace
WORKDIR /go/src/github.com/iovisor/kubectl-trace

RUN make _output/bin/trace-runner

FROM alpine:3.8

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=bpftrace /bpftrace/build-release/src/bpftrace /bin/bpftrace
COPY --from=gobuilder /go/src/github.com/iovisor/kubectl-trace/_output/bin/trace-runner /bin/trace-runner

ENTRYPOINT ["/bin/trace-runner"]

