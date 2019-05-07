FROM alpine:3.8 as builder
ARG bpftracesha
ARG bccversion
ENV STATIC_LINKING=ON
ENV RUN_TESTS=0
RUN apk add --update \
  bison \
  build-base \
  clang-dev \
  clang-static \
  curl \
  cmake \
  elfutils-dev \
  flex-dev \
  git \
  linux-headers \
  llvm5-dev \
  llvm5-static \
  python \
  zlib-dev

# Put LLVM directories where CMake expects them to be
RUN ln -s /usr/lib/cmake/llvm5 /usr/lib/cmake/llvm
RUN ln -s /usr/include/llvm5/llvm /usr/include/llvm
RUN ln -s /usr/include/llvm5/llvm-c /usr/include/llvm-c

WORKDIR /
RUN curl -L https://github.com/iovisor/bcc/archive/v${bccversion}.tar.gz \
  --output /bcc.tar.gz
RUN tar xvf /bcc.tar.gz
RUN mv bcc-${bccversion} bcc
RUN cd /bcc && mkdir build && cd build && cmake .. && make install -j4 && \
  cp src/cc/libbcc.a /usr/local/lib64/libbcc.a && \
  cp src/cc/libbcc-loader-static.a /usr/local/lib64/libbcc-loader-static.a && \
  cp src/cc/libbpf.a /usr/local/lib64/libbpf.a

ADD https://github.com/iovisor/bpftrace/archive/${bpftracesha}.tar.gz /bpftrace.tar.gz
RUN tar -xvf /bpftrace.tar.gz

RUN mv bpftrace-${bpftracesha} /bpftrace

WORKDIR /bpftrace

WORKDIR /bpftrace/docker

RUN sh build.sh /bpftrace/build-release Release bpftrace
