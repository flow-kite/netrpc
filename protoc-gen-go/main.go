// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// protoc-gen-go is a plugin for the Google protocol buffer compiler to generate
// Go code. Install it by building this program and making it accessible within
// your PATH with the name:
//	protoc-gen-go
//
// The 'go' suffix becomes part of the argument for the protocol compiler,
// such that it can be invoked as:
//	protoc --go_out=paths=source_relative:. path/to/file.proto
//
// This generates Go bindings for the protocol buffer defined by file.proto.
// With that input, the output will be written to:
//	path/to/file.pb.go
//
// See the README and documentation for protocol buffers to learn more:
//	https://developers.google.com/protocol-buffers/
package main

import (
	"io/ioutil"
	"os"

	_ "github.com/golang/protobuf/protoc-gen-go/grpc" // 引入grpc插件

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

func main() {
	g := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		g.Error(err, "")
	}

	if err := proto.Unmarshal(data, g.Request); err != nil {
		g.Error(err, "parsing input proto")
	}

	if len(g.Request.FileToGenerate) == 0 {
		g.Fail("no files to generate")
	}

	g.CommandLineParameters(g.Request.GetParameter())

	// Create a wrapped version of the Descriptors and EnumDescriptors that
	// point to the file that defines them.
	g.WrapTypes()

	g.SetPackageNames()
	g.BuildTypeNameMap()

	g.GenerateAllFiles()

	// Send back the results.
	data, err = proto.Marshal(g.Response)
	if err != nil {
		g.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		g.Error(err, "failed to write output proto")
	}
}
