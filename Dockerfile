FROM gcr.io/prysmaticlabs/build-agent

COPY . /src
WORKDIR /src

RUN bazel build //:eth1-mock-rpc

EXPOSE 7777
EXPOSE 7778

ENTRYPOINT ["/src/bazel-bin/linux_amd64_stripped/eth1-mock-rpc"]
