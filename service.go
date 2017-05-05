package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func makeService(modelFile string, serviceFile string) {
	input := modelFile
	output := serviceFile

	fset := token.NewFileSet()                      // positions are relative to fset
	f, err := parser.ParseFile(fset, input, nil, 0) //parser.Trace
	if err != nil {
		fmt.Println(err)
		return
	}
	methods := []Method{}
	// packageName := f.Name.Name
	// ast.Print(fset, f)
	serviceName := ""
	for _, decl := range f.Decls {
		funcDecl, success := decl.(*ast.FuncDecl)
		if !success || funcDecl.Recv == nil {
			continue
		}

		isServiceFunc := false
		for _, field := range funcDecl.Recv.List {
			recPtr, success := field.Type.(*ast.StarExpr)
			if !success {
				continue
			}

			// found a function
			x, success := recPtr.X.(*ast.Ident)
			if !success {
				continue
			}

			// if the receiver pointer ends with 'service' then it must be the type we're looking for
			if !strings.HasSuffix(x.Name, "Service") {
				continue
			}
			serviceName = x.Name

			isServiceFunc = true
		}

		if !isServiceFunc {
			continue
		}
		meth := Method{}
		meth.Name = funcDecl.Name.Name

		ftype := funcDecl.Type
		packageName := "domain"
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
	items := strings.Split(serviceName, "Service")
	domainType := items[0]

	svcInt := ServiceInterface(methods, domainType+"Service")
	repoInt := ServiceInterface(methods, domainType+"Repo")
	cstr := Constructor(domainType+"Repo", domainType+"Service")
	impl := ConformInterface(methods, domainType+"Service")

	serviceOut := fmt.Sprintf("package domain\n%v\n%v\n%v\n%v", svcInt, repoInt, cstr, impl)

	//HACK: instead of figuring out how to not have domain prepend, im just removing all of them after the fact -_-
	serviceOut = strings.Replace(serviceOut, "domain.", "", -1)

	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(serviceOut)
}

func Constructor(repoName string, serviceName string) string {
	constructor := fmt.Sprintf("func New%s(repo %s) *%s\n", serviceName, repoName, serviceName)
	constructor = fmt.Sprintf("%s\ts := new(%s)\n", constructor, serviceName)
	constructor = fmt.Sprintf("%s\ts.repo = repo\n", constructor)
	constructor = fmt.Sprintf("%s\treturn s\n}", constructor)
	return constructor
}

func ConformInterface(methods []Method, serviceName string) string {
	conform := ""
	for _, method := range methods {
		mStr := InterfaceConformance(method, serviceName)
		conform = fmt.Sprintf("%s\n%s", conform, mStr)
	}

	return conform
}

func InterfaceConformance(m Method, serviceName string) string {
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

		ret := ZeroValueFor(r.Kind, r.Type)

		returnsString = append(returnsString, ret)
	}
	returns := strings.Join(returnString, ", ")

	method := fmt.Sprintf("func (s *%s) %s(%s) (%s) {\n", serviceName, m.Name, params, returns)
	method = fmt.Sprintf("%s\treturn s.repo.%s(%s)\n}\n", method, m.Name, calls)

	return method
}

func ServiceInterface(methods []Method, serviceName string) string {
	serviceInterfaceStr := fmt.Sprintf("type I%s interface {\n", serviceName)
	for _, method := range methods {
		iMethodStr := InterfaceMethod(serviceName, method)
		serviceInterfaceStr = fmt.Sprintf("%s\t%s", serviceInterfaceStr, iMethodStr)
	}
	serviceInterfaceStr = fmt.Sprintf("%s}\n\n", serviceInterfaceStr)
	return serviceInterfaceStr
}

func InterfaceMethod(serviceName string, m Method) string {
	paramString := []string{}
	returnString := []string{}
	for _, p := range m.Params {
		paramString = append(paramString, p.Name+" "+p.Type)
	}
	params := strings.Join(paramString, ", ")

	returnsString := []string{}
	for _, r := range m.Returns {
		returnString = append(returnString, r.Type)

		ret := ZeroValueFor(r.Kind, r.Type)

		returnsString = append(returnsString, ret)
	}
	returns := strings.Join(returnString, ", ")

	method := fmt.Sprintf("func %s(%s) (%s)\n", m.Name, params, returns)
	return method
}
