package grpc_http_proxy

import (
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	network_api "github.com/golang/protobuf/ptypes/network/api"
)

// This plugin MUST enable with grpc plugin
const generatedCodeVersion = 1

func init() {
	generator.RegisterPlugin(new(proxy))
}

type proxy struct {
	gen *generator.Generator
}

var (
	contextPkg string
	servePkg   string
)

func (g *proxy) Init(gen *generator.Generator) {
	g.gen = gen
	contextPkg = "context"
	servePkg = generator.RegisterUniquePackageName("github.com/geniuscirno/protobuf-rpc/grpc_http_proxy", nil)
}

func (g *proxy) Name() string {
	return "grpc_http_proxy"
}

func (g *proxy) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

func (g *proxy) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

func (g *proxy) P(args ...interface{}) { g.gen.P(args...) }

func (g *proxy) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("// FBI WARING: Http Service Handler")
	g.P()
	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

func (g *proxy) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName
	}
	servName := generator.CamelCase(origServName)

	var handlerNames []string
	for _, method := range service.Method {
		if !proto.HasExtension(method.Options, network_api.E_Http) {
			continue
		}
		hname := g.generateServerMethod(servName, "", method)
		handlerNames = append(handlerNames, hname)
	}

	clientType := servName + "Client"
	serviceDescVar := "_" + servName + "_grpcHttpProxyServiceDesc"

	g.P("func Register", servName, "GrpcHttpProxyServer(s *", servePkg, ".Server, srv ", clientType, ") {")
	g.P("s.RegisterService(&", serviceDescVar, `, srv)`)
	g.P("}")
	g.P()

	g.P("var ", serviceDescVar, " = ", servePkg, ".ServiceDesc {")
	g.P("ServiceName: ", strconv.Quote(fullServName), ",")
	g.P("HandlerType: (*", clientType, ")(nil),")
	g.P("Methods: []", servePkg, ".MethodDesc{")
	var i int
	for _, method := range service.Method {
		if !proto.HasExtension(method.Options, network_api.E_Http) {
			continue
		}
		ext, err := proto.GetExtension(method.Options, network_api.E_Http)
		if err != nil {
			panic(err)
		}
		var httpMethod, httpPath string
		httpRule := ext.(*network_api.HttpRule)
		switch {
		case httpRule.GetGet() != "":
			httpMethod = "GET"
			httpPath = httpRule.GetGet()
		case httpRule.GetPut() != "":
			httpMethod = "PUT"
			httpPath = httpRule.GetPut()
		case httpRule.GetPost() != "":
			httpMethod = "POST"
			httpPath = httpRule.GetPost()
		case httpRule.GetDelete() != "":
			httpMethod = "DELETE"
			httpPath = httpRule.GetDelete()
		case httpRule.GetPatch() != "":
			httpMethod = "PATCH"
			httpPath = httpRule.GetPatch()
		default:
			panic(fmt.Errorf("no http method match %s", httpRule.Pattern))
		}
		g.P("{")
		g.P("MethodName: ", strconv.Quote(fmt.Sprintf("%s/%s", fullServName, method.GetName())), ",")
		g.P("Handler: ", handlerNames[i], ",")
		g.P("HttpMethod: ", strconv.Quote(httpMethod), ",")
		g.P("HttpPath: ", strconv.Quote(httpPath), ",")
		g.P("},")
		i++
	}
	g.P("},")
	g.P("}")
}

func (g *proxy) generateServerMethod(servName, fullServName string, method *pb.MethodDescriptorProto) string {
	methodName := generator.CamelCase(method.GetName())
	hname := fmt.Sprintf("_%s_%s_GrpcHttpProxyHandler", servName, methodName)
	inType := g.typeName(method.GetInputType())

	g.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, dec func(interface{}) error) (interface{}, error) {")
	g.P("in := new(", inType, ")")
	g.P("if err := dec(in); err != nil {return nil, err }")
	g.P("return srv.(", servName, "Client).", methodName, "(ctx, in)")
	//g.P("info := &", httpPkg, ".ServerInfo{")
	//g.P("Server: srv,")
	//g.P("FullMethod: ", strconv.Quote(methodName), ",")
	//g.P("}")
	//g.P("handler := func(ctx ", contextPkg, ".Context, req interface{}) (interface{}, error) {")
	//g.P("return srv.(", servName, "Client).", methodName, "(ctx, req.(*", inType, "))")
	//g.P("}")
	//g.P("return interceptor(ctx, in, info, handler)")
	g.P("}")
	return hname
}

func (g *proxy) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("import (")
	g.P(servePkg, " ", strconv.Quote("github.com/geniuscirno/protobuf-rpc/grpc_http_proxy"))
	g.P(")")
	g.P()
}
