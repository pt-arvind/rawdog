package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type VariableKind int

const (
	Value VariableKind = iota
	Pointer
	Slice
	Variadic
	GoInterface
)

type Param struct {
	Name string
	Type string
	Kind VariableKind
}

type innerParam struct {
	Param
	Success bool
}

type Method struct {
	Name    string
	Params  []Param
	Returns []Param
}

type Interface struct {
	Methods []Method
	Name    string
	Package string
}

func typeFromField(i interface{}, pkg string) innerParam {
	p := innerParam{}

	// value
	rVType, success := i.(*ast.Ident)
	if success {
		p.Type = rVType.Name
		p.Kind = Value
		p.Success = true
	}

	//slice
	sType, success := i.(*ast.ArrayType)
	if success {
		// elt, success := sType.Elt.(*ast.Ident) //FIXME typeFromField(sType.Elt)
		inner := typeFromField(sType.Elt, pkg)
		if inner.Success {
			p.Type = "[]" + pkg + "." + inner.Type //elt.Name
			p.Kind = Slice
			p.Success = true
		}
	}

	//pointer
	rType, success := i.(*ast.StarExpr)
	if success {
		inner := typeFromField(rType.X, pkg) //rType.X.(*ast.Ident) //FIXME typeFromField(rType.X, pkg)
		if inner.Success {
			p.Type = "*" + pkg + "." + inner.Type
			p.Kind = Pointer
			p.Success = true
		}
	}

	//variadic
	eType, success := i.(*ast.Ellipsis)
	if success {
		//_, success := eType.Elt.(*ast.InterfaceType) //FIXME typeFromField(eType.Elt, pkg)
		inner := typeFromField(eType.Elt, pkg)

		if inner.Success {
			p.Type = "..." + inner.Type
			p.Kind = Variadic
			p.Success = true
		}
	}

	//interface{}
	_, success = i.(*ast.InterfaceType)
	if success {
		p.Type = "interface{}"
		p.Kind = GoInterface
		p.Success = true
	}

	// custom interface i.e. sql.Result
	cType, success := i.(*ast.SelectorExpr)
	if success {
		// x, success := cType.X.(*ast.Ident) //FIXME typeFromField(cType.X, pkg)
		inner := typeFromField(cType.X, pkg)
		if success {
			// p.Type = x.Name + "." + cType.Sel.Name
			p.Type = inner.Type + "." + cType.Sel.Name
			p.Kind = Value
			p.Success = true
		}
	}

	return p
}

func paramFromMember(param *ast.Field, packageName string) []Param {
	ps := []Param{}

	// inner := typeFromField(param.Type, packageName)
	// if len(param.Names) > 0 {
	// 	p.Name = param.Names[0].Name
	// }

	for _, name := range param.Names {
		p := Param{}
		inner := typeFromField(param.Type, packageName)
		if inner.Success {
			p.Name = name.Name
			p.Type = inner.Type
			p.Kind = inner.Kind
		}
		ps = append(ps, p)
	}

	if len(param.Names) == 0 {
		p := Param{}
		inner := typeFromField(param.Type, packageName)
		if inner.Success {
			p.Type = inner.Type
			p.Kind = inner.Kind
		}
		ps = append(ps, p)
	}

	return ps
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: rawdog <file with interfaces> <output file for mocks>")
		return
	}
	input := os.Args[1]
	output := os.Args[2]

	fset := token.NewFileSet()                      // positions are relative to fset
	f, err := parser.ParseFile(fset, input, nil, 0) //parser.Trace
	if err != nil {
		fmt.Println(err)
		return
	}

	packageName := f.Name.Name

	// ast.Print(fset, f)
	allMocks := fmt.Sprintf("// Generated by Rawdog\n")
	for _, decl := range f.Decls {
		root, success := decl.(*ast.GenDecl)

		if !success {
			continue
		}

		interfaceName := ""
		methods := []Method{}

		for _, genDecl := range root.Specs {
			typeSpec, success := genDecl.(*ast.TypeSpec)
			if !success {
				continue
			}
			if typeSpec.Name.Obj.Kind == ast.Typ { // found the interface
				interfaceName = typeSpec.Name.Name
			} else {
				continue
			}

			iface, success := typeSpec.Type.(*ast.InterfaceType)
			if !success {
				continue
			}

			for _, member := range iface.Methods.List {
				meth := Method{}

				if member.Names[0].Obj.Kind == ast.Fun {
					meth.Name = member.Names[0].Name
				}

				ftype, success := member.Type.(*ast.FuncType)
				if !success {
					continue
				}
				for _, param := range ftype.Params.List { //method params
					p := paramFromMember(param, packageName)

					meth.Params = append(meth.Params, p...)
				}
				for _, result := range ftype.Results.List {

					r := paramFromMember(result, packageName)

					meth.Returns = append(meth.Returns, r...)
				}
				methods = append(methods, meth)
			}
		}

		if len(methods) == 0 || interfaceName == "" {
			continue
		}

		i := Interface{}
		i.Methods = methods
		i.Name = interfaceName
		i.Package = packageName

		allMocks = fmt.Sprintf("%s\n\n%s", allMocks, buildMock(i))
	}

	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(allMocks)
	// fmt.Println(allMocks)
}

func buildMock(i Interface) string {
	callbackSuffix := "Callback"
	mockName := "Mock" + i.Name

	structDef := buildStruct(i, mockName, callbackSuffix)
	methodDefs := []string{}
	for _, m := range i.Methods {
		methodDef := buildMethod(m, mockName, callbackSuffix)
		methodDefs = append(methodDefs, methodDef)
	}

	methodDef := strings.Join(methodDefs, "\n")
	resetDef := buildResetMethod(i, mockName, callbackSuffix)

	mockDef := fmt.Sprintf("%s\n\n%s\n%s", structDef, methodDef, resetDef)
	return mockDef
}

func buildStruct(i Interface, mockName string, callbackSuffix string) string {
	structString := fmt.Sprintf("type %s struct {\n", mockName)
	for _, m := range i.Methods {
		paramString := []string{}
		returnString := []string{}
		for _, p := range m.Params {
			paramString = append(paramString, p.Name+" "+p.Type)
		}
		params := strings.Join(paramString, ", ")

		for _, r := range m.Returns {
			returnString = append(returnString, r.Type)
		}
		returns := strings.Join(returnString, ", ")

		method := fmt.Sprintf("%s%s func(%s) (%s)", m.Name, callbackSuffix, params, returns)
		structString = fmt.Sprintf("%s\t%s\n", structString, method)
	}
	structString = fmt.Sprintf("%s}", structString)
	return structString
}

func buildResetMethod(i Interface, mockName string, callbackSuffix string) string {
	method := fmt.Sprintf("func (m *%s) ResetMock() {\n", mockName)
	for _, m := range i.Methods {
		method = fmt.Sprintf("%s\tm.%s%s = nil\n", method, m.Name, callbackSuffix)
	}
	method = fmt.Sprintf("%s}\n", method)
	return method
}

func buildMethod(m Method, mockName string, callbackSuffix string) string {
	paramString := []string{}
	returnString := []string{}
	callString := []string{}
	for _, p := range m.Params {
		paramString = append(paramString, p.Name+" "+p.Type)
		callString = append(callString, p.Name)
	}
	calls := strings.Join(callString, ", ")
	params := strings.Join(paramString, ", ")

	returnsString := []string{}
	for _, r := range m.Returns {
		returnString = append(returnString, r.Type)

		ret := "nil" //for pointers, interfaces
		if r.Kind == Slice {
			ret = "[]"
		}

		returnsString = append(returnsString, ret)
	}
	returns := strings.Join(returnString, ", ")
	returnCalls := strings.Join(returnsString, ", ")

	method := fmt.Sprintf("func (m *%s) %s(%s) (%s) {\n", mockName, m.Name, params, returns)
	method = fmt.Sprintf("%s\tif m.%s%s != nil {\n", method, m.Name, callbackSuffix)
	method = fmt.Sprintf("%s\t\treturn m.%s%s(%s)\n\t}\n", method, m.Name, callbackSuffix, calls)
	method = fmt.Sprintf("%s\treturn %s\n}\n", method, returnCalls)

	return method
}
