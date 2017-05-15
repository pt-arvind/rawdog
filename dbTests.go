package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
)

func makeDBTests(modelFile string, serviceFile string) {
	input := modelFile
	output := serviceFile

	fset := token.NewFileSet()                      // positions are relative to fset
	f, err := parser.ParseFile(fset, input, nil, 0) //parser.Trace
	if err != nil {
		fmt.Println(err)
		return
	}

	//ast.Print(fset, f)
	serviceName := ""
	var dbCols []string
	var varNames []string
	var varTypes []string
	hasAugmented := false
	doneWithObject := false
	for _, decl := range f.Decls {
		genDecl, success := decl.(*ast.GenDecl)

		if success && len(genDecl.Specs) == 1 {
			spec := genDecl.Specs[0]
			typeSpec, success := spec.(*ast.TypeSpec)
			if !success {
				continue
			}

			name := typeSpec.Name.Name
			if strings.HasSuffix(name, "Augmented") {
				hasAugmented = true
			} else {

			}

			structType, success := typeSpec.Type.(*ast.StructType)
			if !success {
				continue
			}

			for _, field := range structType.Fields.List {

				if doneWithObject == false {
					serviceName = name
					if fmt.Sprint(field.Names[0]) == "CreatedAt" {
						doneWithObject = true
						continue
					}

					_, success := field.Type.(*ast.Ident)
					if !success {
						continue
					}
					varName := fmt.Sprint(field.Names[0])
					colName := field.Tag.Value[5 : len(field.Tag.Value)-2]
					dbCols = append(dbCols, colName)
					varNames = append(varNames, varName)
					varTypes = append(varTypes, fmt.Sprint(field.Type))
				}
			}
		}

	}
	domainType := serviceName

	tableName := path.Base(modelFile)
	tableName = tableName[0 : len(tableName)-3]
	getAll := AllTest(domainType, tableName)
	byID := ByIDTest(domainType, tableName, varNames)
	var allAugmented string
	var byIDAugmented string
	var byForeignKey string
	var byForeignKeyAugmented string

	if hasAugmented {
		allAugmented = AllAugmentedTest(domainType, tableName)
		byIDAugmented = ByIDAugmentedTest(domainType, tableName, varNames)
		byForeignKey = ByForeignKeyTests(domainType, tableName, dbCols, varNames)
		byForeignKeyAugmented = ByForeignKeyAugmentedTests(domainType, tableName, dbCols, varNames)
	}

	store := StoreTest(domainType, tableName, varNames, varTypes)
	deleteByID := DeleteByIDTest(domainType, tableName)

	fileHeader := fmt.Sprintf(`package mysqlrepo_test

import (
	"fmt"
	"testing"

	"adapter/mysqlrepo"
	"domain"

	"github.com/stretchr/testify/assert"
)

// Test%sRepo tests the account repo.
func Test%sRepo(t *testing.T) {
	s := mysqlrepo.New%sRepo(sharedDB)`, serviceName, serviceName, serviceName)

	serviceOut := fmt.Sprintf("%v\n\n%v\n\n%v\n\n%v%v%v%v%v\n\n%v\n}", fileHeader, getAll, byID, allAugmented, byIDAugmented, byForeignKey, byForeignKeyAugmented, store, deleteByID)

	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(serviceOut)
}

func AllTest(serviceName, tableName string) string {
	allTestBlock := fmt.Sprintf("\t// Get all %s records in the database.", serviceName)
	allTestBlock = fmt.Sprintf(`%s
	all%s, err := s.All()
	assert.Equal(t, err, nil)
		`, allTestBlock, serviceName)
	return allTestBlock
}

func ByIDTest(serviceName, tableName string, varNames []string) string {
	byIDTestBlock := fmt.Sprintf("\t// Get first %s record by ID.", serviceName)
	byIDTestBlock = fmt.Sprintf(`%s
	item0, err := s.ByID(fmt.Sprint(all%s[0].ID))
	assert.Equal(t, err, nil)
	assert.Equal(t, all%s[0].%s, item0.%s)
		`, byIDTestBlock, serviceName, serviceName, varNames[1], varNames[1])
	return byIDTestBlock
}

func AllAugmentedTest(serviceName, tableName string) string {
	allAugmentedTestBlock := fmt.Sprintf("\t// Get all augmented %s records in the database.", serviceName)
	allAugmentedTestBlock = fmt.Sprintf(`%s
	all%sAugmented, err := s.AllAugmented()
	assert.Equal(t, err, nil)
		`, allAugmentedTestBlock, serviceName)
	return allAugmentedTestBlock
}

func ByIDAugmentedTest(serviceName, tableName string, varNames []string) string {
	byIDAugmentedTestBlock := fmt.Sprintf("\n\t// Get first augmented %s record by ID.", serviceName)
	byIDAugmentedTestBlock = fmt.Sprintf(`%s
	augItem0, err := s.ByIDAugmented(fmt.Sprint(all%sAugmented[0].ID))
	assert.Equal(t, err, nil)
	assert.Equal(t, all%sAugmented[0].%s, augItem0.%s)
		`, byIDAugmentedTestBlock, serviceName, serviceName, varNames[1], varNames[1])
	return byIDAugmentedTestBlock
}

func ByForeignKeyTests(serviceName, tableName string, dbCols, varNames []string) string {
	var byForeignKeyTestsBlock string
	//var foreignKeyList []string
	var foreignKeyVarList []string
	for i, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" {
			foreignKeyVarList = append(foreignKeyVarList, varNames[i])
		}
	}
	for _, foreignKeyVar := range foreignKeyVarList {
		byForeignKeyTestsBlock = fmt.Sprintf("%s\n\n\t// Get %s record by %s.", byForeignKeyTestsBlock, serviceName, foreignKeyVar)

		byForeignKeyTestsBlock = fmt.Sprintf(`%s
	itemsBy%s, err := s.By%s(fmt.Sprint(all%s[0].%s))
	assert.Equal(t, err, nil)
	assert.Equal(t, all%s[0].%s, itemsBy%s[0].%s)
		`, byForeignKeyTestsBlock, foreignKeyVar, foreignKeyVar, serviceName, foreignKeyVar, serviceName, foreignKeyVar, foreignKeyVar, foreignKeyVar)

	}
	return byForeignKeyTestsBlock
}

func ByForeignKeyAugmentedTests(serviceName, tableName string, dbCols, varNames []string) string {
	var byForeignKeyAugmentedTestsBlock string
	//var foreignKeyList []string
	var foreignKeyVarList []string
	for i, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" {
			foreignKeyVarList = append(foreignKeyVarList, varNames[i])
		}
	}
	for _, foreignKeyVar := range foreignKeyVarList {
		byForeignKeyAugmentedTestsBlock = fmt.Sprintf("%s\n\n\t// Get augmented %s record by %s.", byForeignKeyAugmentedTestsBlock, serviceName, foreignKeyVar)

		byForeignKeyAugmentedTestsBlock = fmt.Sprintf(`%s
	augItemsBy%s, err := s.By%sAugmented(fmt.Sprint(all%sAugmented[0].%s))
	assert.Equal(t, err, nil)
	assert.Equal(t, all%sAugmented[0].%s, augItemsBy%s[0].%s)
		`, byForeignKeyAugmentedTestsBlock, foreignKeyVar, foreignKeyVar, serviceName, foreignKeyVar, serviceName, foreignKeyVar, foreignKeyVar, foreignKeyVar)

	}
	return byForeignKeyAugmentedTestsBlock
}

func StoreTest(serviceName, tableName string, varNames, varTypes []string) string {
	byIDTestBlock := fmt.Sprintf("\n\t// Store a %s record.", serviceName)
	var fieldVals string
	for i, varName := range varNames {
		if varName != "ID" {
			typeValue := ""
			if varTypes[i] == "int" {
				typeValue = "1"
			} else if varTypes[i] == "float64" {
				typeValue = "1000.0"
			} else if varTypes[i] == "string" {
				typeValue = fmt.Sprintf(`"test%s"`, varName)
			} else if varTypes[i] == "bool" {
				typeValue = "false"
			}

			fieldVals = fmt.Sprintf(`%s
	%s.%s = %s`, fieldVals, tableName, varName, typeValue)

		}
	}

	byIDTestBlock = fmt.Sprintf(`%s
	%sRepo := mysqlrepo.New%sRepo(sharedDB)
	%s := new(domain.%s)
	%s

	new%s, err := %sRepo.Store(%s)
	assert.Equal(t, err, nil)
	assert.Equal(t, new%s.%s, %s.%s)
		`, byIDTestBlock, tableName, serviceName, tableName, serviceName, fieldVals, serviceName, tableName, tableName, serviceName, varNames[1], tableName, varNames[1])

	return byIDTestBlock
}

func DeleteByIDTest(serviceName, tableName string) string {
	deleteByIDTestBlock := fmt.Sprintf("\t// Delete a %s record by its id.", serviceName)
	deleteByIDTestBlock = fmt.Sprintf(`%s
	err = s.DeleteByID(fmt.Sprint(new%s.ID))
	assert.Equal(t, err, nil)
		`, deleteByIDTestBlock, serviceName)
	return deleteByIDTestBlock
}
