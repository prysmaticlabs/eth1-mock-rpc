#!/bin/bash

NUM_VALIDATORS=$1
DIR=`pwd`/keystore

if [ ! -z "$NUM_VALIDATORS" ];then
	echo "Deleting old validator credentials"
	rm -rf $DIR
	mkdir -p $DIR

	echo "Generate $NUM_VALIDATORS new validator credentials"
	for (( i=1; i<=$NUM_VALIDATORS; i++ ))
	do
     		cd ../prysm
     		bazel run //validator -- accounts create --password 1 --keystore-path $DIR
	done
fi
echo "Generating yaml file"
bazel run //:eth1-mock-rpc -- --password 1 --keystore-path $DIR --output `pwd`/depositsAndKeys.yaml
