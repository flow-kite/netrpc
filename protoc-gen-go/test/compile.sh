#!/bin/zsh


protoc -I . proto/*.proto --go_out=plugins=grpc:.

