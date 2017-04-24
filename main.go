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
	allMocks := ""
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
					p := Param{}
					p.Name = param.Names[0].Name

					// value
					rVType, success := param.Type.(*ast.Ident)
					if success {
						p.Type = rVType.Name
						p.Kind = Value
					}

					// slice
					sType, success := param.Type.(*ast.ArrayType)
					if success {
						elt, success := sType.Elt.(*ast.Ident)
						if success {
							p.Type = "[]" + packageName + "." + elt.Name
							p.Kind = Slice
						}
					}

					//pointer
					rType, success := param.Type.(*ast.StarExpr)
					if success {
						pType, success := rType.X.(*ast.Ident)
						if success {
							p.Type = "*" + packageName + "." + pType.Name
							p.Kind = Pointer
						}

					}

					//variadic
					eType, success := param.Type.(*ast.Ellipsis)
					if success {
						_, success := eType.Elt.(*ast.InterfaceType) //TODO: this should really use a generic function that accepts all possibilities
						if success {
							p.Type = "...interface{}"
							p.Kind = Variadic
						}
					}

					//interface{}
					_, success = param.Type.(*ast.InterfaceType)
					if success {
						p.Type = "interface{}"
						p.Kind = GoInterface
					}

					meth.Params = append(meth.Params, p)
				}
				for _, result := range ftype.Results.List {
					r := Param{}

					// pointer?
					rType, success := result.Type.(*ast.StarExpr)
					if success {
						pType, success := rType.X.(*ast.Ident)
						if !success {
							continue
						}
						r.Type = "*" + packageName + "." + pType.Name
					}

					// value
					rVType, success := result.Type.(*ast.Ident)
					if success {
						r.Type = rVType.Name
					}

					// slice
					sType, success := result.Type.(*ast.ArrayType)
					if success {
						elt, success := sType.Elt.(*ast.Ident)
						if success {
							r.Type = "[]" + packageName + "." + elt.Name
						}
					}

					// custom interface
					cType, success := result.Type.(*ast.SelectorExpr)
					if success {
						x, success := cType.X.(*ast.Ident)
						if success {
							r.Type = x.Name + "." + cType.Sel.Name
						}
					}

					meth.Returns = append(meth.Returns, r)
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
		fmt.Println("ERROR: %v", err)
		return
	}
	defer file.Close()

	file.WriteString(allMocks)
}

func buildMock(i Interface) string {
	callbackSuffix := "Callback"
	mockPrefix := "Mock" + i.Name

	structDef := buildStruct(i, mockPrefix, callbackSuffix)
	methodDefs := []string{}
	for _, m := range i.Methods {
		methodDef := buildMethod(m, mockPrefix, callbackSuffix)
		methodDefs = append(methodDefs, methodDef)
	}

	methodDef := strings.Join(methodDefs, "\n")

	mockDef := fmt.Sprintf("%s\n\n%s", structDef, methodDef)
	return mockDef
}

func buildStruct(i Interface, mockPrefix string, callbackSuffix string) string {
	structString := fmt.Sprintf("type %s struct {\n", mockPrefix)
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
