FROM gcr.io/prysmaticlabs/build-agent as builder

COPY . /src
WORKDIR /src

RUN bazel build //:eth1-mock-rpc

EXPOSE 7777
EXPOSE 7778

FROM gcr.io/whiteblock/base:ubuntu1804

COPY --from=builder /src/bazel-bin/linux_amd64_stripped/eth1-mock-rpc /usr/local/bin/


ENTRYPOINT ["/usr/local/bin/eth1-mock-rpc"]
