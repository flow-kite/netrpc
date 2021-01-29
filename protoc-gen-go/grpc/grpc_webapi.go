package grpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/o-kit/micro-kit/dist/proto/common"

	"github.com/o-kit/netrpc/proto"
	pb "github.com/o-kit/netrpc/protoc-gen-go/descriptor"
	"github.com/o-kit/netrpc/protoc-gen-go/generator"
)

// TODO 4. GenWebApi 相关内容
func (g *grpc) GenWebApi(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto) {
	g.GenMService(file, service)
	g.P("type ", service.Name, "WebApiRegister interface {")
	g.P(service.Name, "Server")
	g.P(mservicePkg, ".WebApiRegister")
	g.P("}")
	g.P()

	g.P("func Register", service.Name, "WebApi(s ", service.Name, "WebApiRegister) {")
	g.P("Register", service.Name, "WebApiImpl(s, s)")
	g.P("}")
	g.P()

	g.P("func Register", service.Name, "WebApiImpl(s ", mservicePkg, ".WebApiRegister, impl ", service.Name, "Server) {")
	pkg := file.GetPackage()
	if pkg != "" {
		pkg += "."
	}
	g.P("wrap := &", service.Name, "WebApi{server: impl, register: s}")
	for _, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		path := fmt.Sprintf("/api/%v%v/%v", pkg, service.GetName(), method.GetName())
		g.P("s.WebApiRegister(", strconv.Quote(path), ", wrap.", method.Name, ")")
	}
	g.P("}")
	g.P()

	g.P("func Register", service.Name, "WebApiEx(s ", service.Name, "WebApiRegister) {")
	g.P("wrap := &", service.Name, "WebApi{server: s, register: s}")
	pkg = file.GetPackage()
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
	// TODO by zzj , 这里后期可能会添加serviceOpDesc
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
	g.P("Register", service.Name, "WebApiEx(s)")
	g.P("}")
	g.P()
}

// 这里是一个大类 - 包含了所有我们自定义的扩展内容，通过 proto.ExtensionDesc
// 这里主要是针对GRPC和WebApi的一些参数与限制进行自定义
func (g *grpc) GenDesc(file *generator.FileDescriptor) {
	for _, service := range file.GetService() {
		var desc common.ServiceOpDesc
		pkg := file.GetPackage()
		if pkg != "" {
			pkg += "."
		}
		desc.Name = pkg + service.GetName()

		// 设置service级别扩展
		if op := getExtension(service.Options, common.E_Service); op != nil {
			desc.Option = op.(*common.ServiceOption)
		}

		desc.Methods = make([]*common.MethodOpDesc, len(service.GetMethod()))
		for idx, method := range service.GetMethod() {
			opDesc := &common.MethodOpDesc{
				Name: method.GetName(),
			}
			desc.Methods[idx] = opDesc
			if method.ServerStreaming != nil && *method.ServerStreaming {
				opDesc.IsServerStreaming = true
			}
			if method.ClientStreaming != nil && *method.ClientStreaming {
				opDesc.IsClientStreaming = true
			}

			if op := getExtension(method.Options, common.E_Webapi); op != nil {
				opDesc.Webapi = op.(*common.WebapiOption)
			}
			if op := getExtension(method.Options, common.E_Auth); op != nil {
				opDesc.Auth = op.(*common.AuthOption)
			}

			// 设置每个方法的过期时间
			if op := getExtension(method.Options, common.E_Timeout); op != nil {
				var timeout string
				if timeoutPtr := op.(*string); timeoutPtr != nil {
					timeout = *timeoutPtr
				}

				switch timeout {
				case "-1":
					opDesc.Timeout = &common.Duration{Nanos: -1}
				default:
					duration, err := time.ParseDuration(timeout)
					if err != nil {
						panic("invalid duration")
					}
					opDesc.Timeout = &common.Duration{Nanos: int64(duration)}
				}
			}
		}
		n, _ := proto.Marshal(&desc)
		g.P("var Option", service.Name, " = common.GenOption([]byte{")
		g.getBin(n, "Option")
		g.P("})")
	}
}

func getExtension(p proto.Message, extension *proto.ExtensionDesc) interface{} {
	if p == (*pb.ServiceOptions)(nil) {
		return nil
	}
	if p == (*pb.MethodOptions)(nil) {
		return nil
	}
	if !proto.HasExtension(p, extension) {
		return nil
	}
	obj, _ := proto.GetExtension(p, extension)
	return obj
}

func (g *grpc) getBin(b []byte, common string) {
	g.gen.In()
	g.gen.P("// ", len(b), "bytes of ", common)
	for len(b) > 0 {
		n := 16
		if n > len(b) {
			n = len(b)
		}

		s := ""
		for _, c := range b[:n] {
			s += fmt.Sprintf("0x%02x,", c)
		}
		g.P(s)
		b = b[n:]
	}
	g.gen.Out()
}
