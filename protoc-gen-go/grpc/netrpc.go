package grpc

import (
	"fmt"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

func init() {
	generator.RegisterPlugin(new(netrpcPlugin))
}

type netrpcPlugin struct {
	gen *generator.Generator
}

func (p *netrpcPlugin) Name() string {
	return "netrpc"
}

func (p *netrpcPlugin) Init(g *generator.Generator) {
	p.gen = g
}

func (p *netrpcPlugin) P(args ...interface{}) {
	p.gen.P(args...)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (p *netrpcPlugin) objectNamed(name string) generator.Object {
	p.gen.RecordTypeUse(name)
	return p.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (p *netrpcPlugin) typeName(str string) string {
	return p.gen.TypeName(p.objectNamed(str))
}

func (p *netrpcPlugin) GenerateImports(file *generator.FileDescriptor) {
	if len(file.Service) > 0 {
		p.genImportCode(file)
	}
}

func (p *netrpcPlugin) Generate(file *generator.FileDescriptor) {

	contextPkg = string(p.gen.AddImport(contextPkgPath))
	grpcPkg = string(p.gen.AddImport(grpcPkgPath))

	for i, svc := range file.FileDescriptorProto.Service {
		p.generateService(file, svc, i)
	}
}

// 生成导入代码 import
func (p *netrpcPlugin) genImportCode(file *generator.FileDescriptor) {
	p.P("import (")
	p.P(`"context"`)
	p.P(")")
	p.P()
}

// 为每个服务生成代码
func (p *netrpcPlugin) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}

	servName := generator.CamelCase(origServName)
	deprecated := service.GetOptions().GetDeprecated()

	p.P()
	// Server interface.
	serverType := servName + "Server"
	p.P("// ", serverType, " is the server API for ", servName, " service.")
	if deprecated {
		p.P("//")
		p.P(deprecationComment)
	}
	p.P("type ", serverType, " interface {")
	for i, method := range service.Method {
		p.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		p.P(p.generateServerSignature(servName, method))
	}
	p.P("}")
	p.P()

}

func (p *netrpcPlugin) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	reqArg := ", in *" + p.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "*" + p.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx %s.Context%s, opts ...%s.CallOption) (%s, error)", methName, contextPkg, reqArg, grpcPkg, respName)
}

// generateServerSignature returns the server-side signature for a method.
func (p *netrpcPlugin) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, contextPkg+".Context")
		ret = "(*" + p.typeName(method.GetOutputType()) + ", error)"
	}
	if !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "*"+p.typeName(method.GetInputType()))
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
		reqArgs = append(reqArgs, servName+"_"+generator.CamelCase(origMethName)+"Server")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}
