# Eth1 Mock RPC

[![Discord](https://user-images.githubusercontent.com/7288322/34471967-1df7808a-efbb-11e7-9088-ed0b04151291.png)](https://discord.gg/KSA7rPr)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/prysmaticlabs/geth-sharding?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

This is a Go tool that mocks the basics of Ethereum 1.0 RPC server for usage as an endpoint for Ethereum 2.0. Created by the Prysmatic Labs team, building a production client for Ethereum 2.0 called [Prysm](https://github.com/prysmaticlabs/prysm). **WARNING**: This is _NOT_ a generic Ethereum mock RPC server, as it's only purpose is to serve Ethereum 2.0 clients.

### Why do we need this?

Ethereum 2.0 is an entirely new blockchain protocol which will bring much needed scalability and security upgrades to the current Ethereum ecosystem. Contrary to being a hard fork, Ethereum 2.0 will be a separate system built from scratch running [Proof of Stake](https://github.com/ethereum/wiki/wiki/Proof-of-Stake-FAQ) consensus. Participants in consensus are known as **validators**, and they join the network by depositing 32 ETH into a **validator deposit contract** deployed on the current Ethereum Proof of Work chain. Nodes running Ethereum 2.0 need to listen to these deposit contract events in order to kick-off the chain and onboard new validators.

This project serves as a mock server that simulates that deposit functionality without the need to run a real Ethereum network, making it easier to run local testnets for Ethereum 2.0. It is meant to be used alongside an Ethereum 2.0 client such as [Prysm](https://github.com/prysmaticlabs/prysm).

## Installation

To run the tool, you'll need to install: 

  - The latest release of [Bazel](https://docs.bazel.build/versions/master/install.html)
  - A modern GNU/Linux operating system

### Build the Mock ETH1 RPC Server

1. Open a terminal window. Ensure you are running the most recent version of Bazel by issuing the command:
```
bazel version
```
2. Clone this repository and enter the directory:
```
git clone https://github.com/prysmaticlabs/eth1-mock-rpc
cd prysm
```
3. Build the project:
```
bazel build ///...
```
Bazel will automatically pull and install any dependencies as well, including Go and necessary compilers.

### Build the Prysm Project

Instructions on downloading and building Prysm for local use can be found in the latest Prysm [README.md](https://github.com/prysmaticlabs/prysm/blob/master/README.md).


## Running the Mock ETH1 RPC Server

### Generate Validator Keys

You'll need to generate a bunch of validator private keys so that the mock ETH1 server can simulate enough validator deposits. **Navigate to the Prysm repository locally**, and run:

```sh
bazel run //validator -- accounts create --password PASSWORD --keystore-path /path/to/keystore
```

This will generate a single validator in your local keystore. You'll need 64 minimum to reach genesis in Prysm. Now, using that same keystore you just created, **in a separate terminal window**, navigate to the `eth1-mock-rpc` project and launch the eth1 mock RPC:

```sh
bazel run //:eth1-mock-rpc -- --password PASSWORD --keystore-path /path/to/keystore
```

Once your server is running, it will launch an HTTP and websocket listener at http://localhost:7777 and http://localhost:7778 respectively. You can now launch the Prysm project and point it to these endpoints to receive mock data:

```sh
bazel run //beacon-chain -- \
--no-discovery \
--http-web3provider http://localhost:7777 \
--web3provider ws://localhost:7778 \
--clear-db \
--verbosity debug
```

## License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0.html)


