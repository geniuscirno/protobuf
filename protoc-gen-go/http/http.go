package http

import (
	"fmt"
	"path"
	"strconv"

	"proto/network/api"

	"github.com/golang/protobuf/proto"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

const generatedCodeVersion = 1

const (
	contextPkgPath = "context"
)

func init() {
	generator.RegisterPlugin(new(http))
}

type http struct {
	gen *generator.Generator
}

var (
	contextPkg string
)

func (g *http) Init(gen *generator.Generator) {
	g.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
}

func (g *http) Name() string {
	return "http"
}

func (g *http) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

func (g *http) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

func (g *http) P(args ...interface{}) { g.gen.P(args...) }

func (g *http) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ ", contextPkg, ".Context")
	g.P()

	for i, service := range file.FileDescriptorProto.Service {
		found := false
		//因为所有proto依赖的文件都会放到FileDescriptorProto,所以需要判断一下file是不是需要生成的
		for _, f := range g.gen.Request.GetFileToGenerate() {
			if f == file.GetName() {
				found = true
			}
		}
		if found {
			g.generateService(file, service, i)
		}
	}
}

func (g *http) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	origServName := service.GetName()
	servName := generator.CamelCase(origServName)

	g.P("// Server API for ", servName, " service")
	g.P()

	serverType := servName + "Server"
	g.P()
	g.P("type ", serverType, " interface{")
	for _, method := range service.Method {
		g.P(g.generateServerSignature(servName, method))
	}
	g.P("}")

	g.P("func Register", servName, "Server(srv ", serverType, "){")
	for _, method := range service.Method {
		g.generateServerMethod(file, servName, method)
	}
	g.P("}")
}

func (g *http) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methodName := generator.CamelCase(origMethName)

	reqName := ", *" + g.typeName(method.GetInputType())
	respName := "*" + g.typeName(method.GetOutputType())
	return fmt.Sprintf("%s(%s.Context%s)(%s, error)", methodName, contextPkg, reqName, respName)
}

func parseHttpRule(file *generator.FileDescriptor, method *pb.MethodDescriptorProto) (string, string) {
	ext, err := proto.GetExtension(method.Options, network_api.E_Http)
	if err != nil {
		panic(fmt.Errorf("%s,%s:%s", file.GetName(), method.GetName(), err))
	}

	var (
		httpMethod string
		httpPath   string
	)

	rule := ext.(*network_api.HttpRule)
	switch {
	case rule.GetGet() != "":
		httpMethod = "GET"
		httpPath = rule.GetGet()
	case rule.GetPut() != "":
		httpMethod = "PUT"
		httpPath = rule.GetPut()
	case rule.GetPost() != "":
		httpMethod = "POST"
		httpPath = rule.GetPost()
	case rule.GetDelete() != "":
		httpMethod = "DELETE"
		httpPath = rule.GetDelete()
	case rule.GetPatch() != "":
		httpMethod = "PATCH"
		httpPath = rule.GetPatch()
	default:
		panic(fmt.Errorf("no http method match %s", rule.Pattern))
	}
	return httpMethod, httpPath
}

func (g *http) generateServerMethod(file *generator.FileDescriptor, servName string, method *pb.MethodDescriptorProto) {
	httpMethod, httpPath := parseHttpRule(file, method)
	methodName := generator.CamelCase(method.GetName())
	inType := g.typeName(method.GetInputType())

	g.P("http.HandleFunc(", strconv.Quote(httpPath), ", common.HttpErrorHandler(func (w http.ResponseWriter, r *http.Request) error {")
	g.P("if r.Method != ", strconv.Quote(httpMethod), `{return errors.New("invalid http method")}`)
	g.P("req := &", inType, "{}")
	g.P("if r.Header.Get(\"Content-Type\")==\"application/json\"{")
	//g.P("if _,err := r.GetBody(); err != nil {return err}")
	g.P("if err :=  json.NewDecoder(r.Body).Decode(req); err != nil {return err}")
	g.P("}else{")
	g.P("if err := r.ParseForm(); err != nil {return err}")
	g.P(`if err := schema.NewDecoder().Decode(req, r.PostForm); err != nil {return err}}`)
	g.P("log.Println(", strconv.Quote(httpPath), ", req)")
	g.P("reply, err := srv.", methodName, "(", contextPkg, ".TODO(), req)")
	g.P("if err != nil {return err}")
	g.P("return json.NewEncoder(w).Encode(reply)")
	g.P("}))")
}

func (g *http) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("import (")
	g.P(contextPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, contextPkgPath)))
	g.P(strconv.Quote("encoding/json"))
	g.P(strconv.Quote("net/http"))
	g.P(strconv.Quote("github.com/gorilla/schema"))
	g.P(strconv.Quote("log"))
	g.P(strconv.Quote("errors"))
	g.P(strconv.Quote("common"))
	g.P(")")
	g.P()
}
