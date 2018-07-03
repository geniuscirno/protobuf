package message

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

const generatedCodeVersion = 1

const (
	contextPkgPath = "context"
	msgPkgPath     = "github.com/geniuscirnp/protobuf-rpc/message"
)

func init() {
	generator.RegisterPlugin(new(message))
}

type message struct {
	gen *generator.Generator
}

func (g *message) Name() string {
	return "message"
}

var (
	contextPkg string
	rpcPkg     string
)

func (g *message) Init(gen *generator.Generator) {
	g.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	rpcPkg = generator.RegisterUniquePackageName("msg", nil)
}

func (g *message) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

func (g *message) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

func (g *message) P(args ...interface{}) { g.gen.P(args...) }

func (g *message) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ ", contextPkg, ".Context")
	g.P()

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// func getExtendApiId(method *pb.MethodDescriptorProto) *uint32 {
// 	ext, err := proto.GetExtension(method.Options, network_api.E_Id)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return ext.(*uint32)
// }

// func hasExtengNotify(method *pb.MethodDescriptorProto) bool {
// 	return proto.HasExtension(method.Options, network_api.E_Notify)
// }

func getInTypeEnum(file *generator.FileDescriptor, typeName string) *int32 {
	for _, e := range file.GetEnumType() {
		if e.GetName() == "api" {
			for _, v := range e.GetValue() {
				if v.GetName() == "E_"+typeName {
					return v.Number
				}
			}
		}
	}
	panic(fmt.Errorf("cannot find bind inType %s's id", typeName))
}

func (g *message) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	serviceDescVar := "_" + servName + "_serviceDesc"

	g.P("// Server API for ", servName, " service")
	g.P()

	serverType := servName + "Server"

	g.P("// Request")
	g.P("type ", serverType, " interface {")
	for _, method := range service.Method {
		g.P(g.generateServerSignature(servName, method))
	}
	g.P("}")
	g.P()

	// g.P("// Notify")
	// g.P("type ", servName, "Notify interface{")
	// for _, method := range service.Method {
	// 	if hasExtengNotify(method) {
	// 		g.P(g.generateServerNotifySignature(servName, method))
	// 	}
	// }
	// g.P("}")
	// g.P()

	// // Notify structure
	// g.P("type ", unexport(servName), "Notify struct{ ")
	// g.P("ctx ", contextPkg, ".Context")
	// g.P("}")
	// g.P()

	// //NewClient factory
	// g.P("func New", servName, "Notify (ctx ", contextPkg, ".Context) ", servName, "Notify{")
	// g.P("return &", unexport(servName), "Notify{ctx: ", "ctx}")
	// g.P("}")
	// g.P()

	// //Notify method implement
	// for _, method := range service.Method {
	// 	if hasExtengNotify(method) {
	// 		g.generateServerNotifyMethod(servName, method)
	// 	}
	// }

	// Server registratioon
	g.P("func Register", servName, "Server(s *", rpcPkg, ".Server, srv ", serverType, "){ ")
	g.P("s.RegisterService(&", serviceDescVar, ", srv)")
	g.P("}")
	g.P()

	// Server handler implement
	var handlerNames []string
	for _, method := range service.Method {
		hname := g.generateServerMethod(servName, method)
		handlerNames = append(handlerNames, hname)
	}

	// Service descriptor.
	g.P("var ", serviceDescVar, " = ", rpcPkg, ".ServiceDesc {")
	g.P("ServiceName: ", strconv.Quote(servName), ",")
	g.P("HandlerType: (*", serverType, ")(nil),")
	g.P("Methods: []", rpcPkg, ".MethodDesc{")
	for i, method := range service.Method {
		g.P("{")
		g.P("MethodName: ", strconv.Quote(method.GetName()), ",")
		inType := g.typeName(method.GetInputType())
		g.P("MethodId: ", getInTypeEnum(file, inType), ",")
		g.P("Handler: ", handlerNames[i], ",")
		g.P("},")
	}
	g.P("},")
	//g.P("Notify: []", rpcPkg, ".NotifyDesc{")
	//for i, method := range service.Method {
	//	if hasExtengNotify(method) {
	//		g.P("{")
	//		g.P("NotifyName: ", strconv.Quote(method.GetName()), ",")
	//		ext, err := proto.GetExtension(method.Options, network_api.E_Id)
	//		if err != nil {
	//			panic(err)
	//		}
	//		g.P("NotifyId": getExtendApiId(method), ",")
	//	}
	//}
	g.P("}")
	g.P()
}

func (g *message) generateServerMethod(servName string, method *pb.MethodDescriptorProto) string {
	methName := generator.CamelCase(method.GetName())
	hname := fmt.Sprintf("_%s_%s_Handler", servName, methName)
	inType := g.typeName(method.GetInputType())

	g.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, dec func (interface{}) error) (interface{}, error) {")
	g.P("in := new(", inType, ")")
	g.P("if err := dec(in); err != nil {return nil, err}")
	g.P("return srv.(", servName, "Server).", methName, "(ctx, in )")
	g.P("}")
	return hname
}

func (g *message) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methodName := generator.CamelCase(origMethName)

	reqArg := ", *" + g.typeName(method.GetInputType())
	respName := "*" + g.typeName(method.GetOutputType())
	return fmt.Sprintf("%s(%s.Context%s)(%s, error)", methodName, contextPkg, reqArg, respName)
}

func (g *message) generateServerNotifySignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methodName := generator.CamelCase(origMethName)

	reqArg := "in *" + g.typeName(method.GetInputType())
	respName := "*" + g.typeName(method.GetOutputType())
	return fmt.Sprintf("%s(%s)(%s, error)", methodName, reqArg, respName)
}

func (g *message) generateServerNotifyMethod(servName string, method *pb.MethodDescriptorProto) {
	g.P("func (c *", unexport(servName), "Notify) ", g.generateServerNotifySignature(servName, method), "{")
	//g.P("err := rpc.Notify(c.ctx, ", getExtendApiId(method), ",in)")
	//g.P("return nil, err")
	g.P("return nil, nil")
	g.P("}")
	g.P()
}

func (g *message) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("import (")
	g.P(contextPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, contextPkgPath)))
	g.P(rpcPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, msgPkgPath)))
	g.P(")")
	g.P()
}
