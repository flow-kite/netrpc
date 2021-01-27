package grpc

import (
	"fmt"
	"strconv"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

// TODO 4. GenWebApi 相关内容
func (g *grpc) GenWebApi(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto) {
	g.GenMService(file, service)
	g.P("type ", service.Name, "WebApiRegister interface {")
	g.P(service.Name, "Server")
	g.P(mservicePkg, ".WebApiRegister")
	g.P("}")
	g.P()

	g.P("func Register", service.Name, "WebApiImpl(s ", service.Name, "WebApiRegister) {")
	g.P("wrap := &", service.Name, "WebApi{server: s, register: s}")
	pkg := file.GetPackage()
	if pkg != "" {
		pkg += "."
	}
	for _, method := range service.Method {
		if method.GetClientStreaming() || method.GetServerStreaming() {
			continue
		}

		path := fmt.Sprintf("/api/%v%v/%v", pkg, service.GetName(), method.GetName())
		g.P("s.WebApiRegister(", strconv.Quote(path), ", wrap.", method.Name, ")")
	}
	g.P("}")
	g.P()

	// 定义webApi的handler - for 每个路由规则
	g.P("type ", service.GetName(), "WebApi struct {")
	g.P("server ", service.GetName(), "Server")
	g.P("register ", mservicePkg, ".WebApiRegister")
	g.P("}")
	g.P()

	// 这里实现 http -> grpc 转换
	for _, method := range service.Method {
		if method.GetClientStreaming() || method.GetServerStreaming() {
			continue
		}
		g.P("func (s *", service.GetName(), "WebApi) ", method.GetName(), "(ctx *context.T, w http.ResponseWriter, req *http.Request) {")
		g.P("params := new(", g.typeName(method.GetInputType()), ")")
		g.P("if err := s.register.WebApiDecode(ctx, req, params); err != nil {")
		g.P("s.register.WebApiHandleResp(ctx, w, nil, err)")
		g.P("return")
		g.P("}")
		g.P("resp, err := s.server.", method.GetName(), "(*ctx, params)")
		g.P("s.register.WebApiHandleResp(ctx, w, resp, err)")
		g.P("}")
		g.P()
	}
}

func (g *grpc) GenMService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto) {
	g.P("type ", service.Name, "MService interface {")
	g.P("RegisterServiceDesc(*common.ServiceOpDesc)")
	g.P(service.Name, "Server")
	g.P(mservicePkg, ".WebApiRegister")
	g.P(mservicePkg, ".GrpcRegister")
	g.P("}")
	g.P()

	g.P("type ", service.Name, "GrpcRegister interface {")
	g.P(service.Name, "Server")
	g.P(mservicePkg, ".GrpcRegister")
	g.P("}")
	g.P()

	g.P("func Register", service.Name, "(s ", service.Name, "MService", ") {")
	g.P("s.RegisterServiceDesc(Option", service.Name, ")")
	g.P("Register", service.Name, "Grpc(s)")
	g.P("Register", service.Name, "WebApiImpl(s)")
	g.P("}")
	g.P()
}

func (g *grpc) GenDesc(file *generator.FileDescriptor) {

}
