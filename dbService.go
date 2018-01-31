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

func makeDBService(modelFile string, serviceFile string) {
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
				}
			}
		}

	}
	domainType := serviceName

	tableName := path.Base(modelFile)
	tableName = tableName[0 : len(tableName)-3]
	getAll := AllQuery(domainType, tableName)
	byID := ByIDQuery(domainType, tableName)
	var allAugmented string
	var byIDAugmented string
	var byForeignKey string
	var byForeignKeyAugmented string
	if hasAugmented {
		allAugmented = AllAugmentedQuery(domainType, tableName, dbCols)
		byIDAugmented = ByIDAugmentedQuery(domainType, tableName, dbCols)
		byForeignKey = ByForeignKeyQueries(domainType, tableName, dbCols, varNames)
		byForeignKeyAugmented = ByForeignKeyAugmentedQueries(domainType, tableName, dbCols, varNames)
	}
	store := StoreQuery(domainType, tableName, dbCols, varNames)
	deleteByID := DeleteByIDQuery(domainType, tableName)

	serviceOut := fmt.Sprintf("// The following text should be inserted after toEntity[Augmented] \n\n%v\n\n%v\n\n%v%v%v%v%v\n\n%v", getAll, byID, allAugmented, byIDAugmented, byForeignKey, byForeignKeyAugmented, store, deleteByID)

	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString(serviceOut)
}
func AllQuery(serviceName, tableName string) string {
	allQueryBlock := fmt.Sprintf("// All will retrieve all %s records in the database.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) All() ([]domain.%s, error) {", serviceName, serviceName)
	methodContents := fmt.Sprintf("\tdb%sRecords := []%s{}\n\terr := s.db.Connection().Select(&db%sRecords, `", serviceName, serviceName, serviceName)
	sqlQuery := fmt.Sprintf(`
		Select %s.*
		FROM %s
		WHERE deleted_at IS NULL
		`, tableName, tableName)
	handleErrStr := fmt.Sprintf("\n\tif err != nil {\n\t\t return nil, err\n\t}")
	appendResultArray := fmt.Sprintf("\tresult := []domain.%s{}\n\tfor _, %s := range db%sRecords {\n\t\tresult = append(result, *%s.toEntity())\n\t}\n\n\treturn result, nil\n}", serviceName, tableName, serviceName, tableName)
	allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`)\n%s\n\n%s", allQueryBlock, methodStr, methodContents, sqlQuery, handleErrStr, appendResultArray)
	return allQueryBlock
}

func ByIDQuery(serviceName, tableName string) string {
	allQueryBlock := fmt.Sprintf("// ByID will retrieve the %s record with the input ID.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) ByID(id string) (*domain.%s, error) {", serviceName, serviceName)
	methodContents := fmt.Sprintf("\tresult := %s{}\n\terr := s.db.Connection().Get(&result, `", serviceName)
	sqlQuery := fmt.Sprintf(`
		Select %s.*
		FROM %s
		WHERE id = ?
			AND deleted_at IS NULL
		LIMIT 1	
		`, tableName, tableName)
	handleReturnStr := fmt.Sprintf("\n\treturn result.toEntity(), err\n}")
	allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`, id)\n%s", allQueryBlock, methodStr, methodContents, sqlQuery, handleReturnStr)
	return allQueryBlock
}

func AllAugmentedQuery(serviceName, tableName string, dbCols []string) string {
	allQueryBlock := fmt.Sprintf("// AllAugmented will retrieve all %sAugmented records in the database.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) AllAugmented() ([]domain.%sAugmented, error) {", serviceName, serviceName)
	methodContents := fmt.Sprintf("\tdb%sRecords := []%sAugmented{}\n\terr := s.db.Connection().Select(&db%sRecords, `", serviceName, serviceName, serviceName)
	var additionalTableStr string
	var joinStr string
	for _, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" { //TODO also make sure it doesn't contaim fromOU or to OU
			joinTable := dbCol[len(tableName)+1 : lenCol-3]
			additionalTableStr = additionalTableStr + ", " + joinTable + ".*"
			joinStr = fmt.Sprintf(`%s
		JOIN %s
			ON %s.%s_id = %s.id
			AND %s.deleted_at IS NULL`, joinStr, joinTable, tableName, joinTable, joinTable, joinTable)
		}
	}
	sqlQuery := fmt.Sprintf(`
		Select %s.*%s
		FROM %s%s
		WHERE %s.deleted_at IS NULL
		`, tableName, additionalTableStr, tableName, joinStr, tableName)
	handleErrStr := fmt.Sprintf("\n\tif err != nil {\n\t\t return nil, err\n\t}")
	appendResultArray := fmt.Sprintf("\tresult := []domain.%sAugmented{}\n\tfor _, %s := range db%sRecords {\n\t\tresult = append(result, *%s.toEntityAugmented())\n\t}\n\n\treturn result,nil\n}", serviceName, tableName, serviceName, tableName)
	allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`)\n%s\n\n%s\n\n", allQueryBlock, methodStr, methodContents, sqlQuery, handleErrStr, appendResultArray)
	return allQueryBlock
}

func ByIDAugmentedQuery(serviceName, tableName string, dbCols []string) string {
	allQueryBlock := fmt.Sprintf("// ByIDAugmented will retrieve the %sAugmented record with the input ID.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) ByIDAugmented(id string) (*domain.%sAugmented, error) {", serviceName, serviceName)
	methodContents := fmt.Sprintf("\tresult := %sAugmented{}\n\terr := s.db.Connection().Get(&result, `", serviceName)
	var additionalTableStr string
	var joinStr string
	for _, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" { //TODO also make sure it doesn't contaim fromOU or to OU
			joinTable := dbCol[len(tableName)+1 : lenCol-3]
			additionalTableStr = additionalTableStr + ", " + joinTable + ".*"
			joinStr = fmt.Sprintf(`%s
		JOIN %s
			ON %s.%s_id = %s.id
			AND %s.deleted_at IS NULL`, joinStr, joinTable, tableName, joinTable, joinTable, joinTable)
		}
	}
	sqlQuery := fmt.Sprintf(`
		Select %s.*%s
		FROM %s%s
		WHERE %s.id = ?
			AND %s.deleted_at IS NULL
		LIMIT 1	
		`, tableName, additionalTableStr, tableName, joinStr, tableName, tableName)
	handleReturnStr := fmt.Sprintf("\n\treturn result.toEntityAugmented(), err\n}")
	allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`, id)\n%s\n\n", allQueryBlock, methodStr, methodContents, sqlQuery, handleReturnStr)
	return allQueryBlock
}

func ByForeignKeyQueries(serviceName, tableName string, dbCols, varNames []string) string {
	var foreignKeyList []string
	var foreignKeyVarList []string
	var byForeignKeyQueriesStr string

	for i, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" {
			foreignKey := dbCol[len(tableName)+1:]
			foreignKeyList = append(foreignKeyList, foreignKey)
			foreignKeyVarList = append(foreignKeyVarList, varNames[i])
		}
	}

	for i, foreignKey := range foreignKeyList {
		allQueryBlock := fmt.Sprintf("// By%s will retrieve all %s records in the database with a given %s.", foreignKeyVarList[i], serviceName, foreignKey)
		methodStr := fmt.Sprintf("func (s *%sService) By%s(%s string) ([]domain.%s, error) {", serviceName, foreignKeyVarList[i], foreignKey, serviceName)
		methodContents := fmt.Sprintf("\tdb%sRecords := []%s{}\n\terr := s.db.Connection().Select(&db%sRecords, `", serviceName, serviceName, serviceName)
		sqlQuery := fmt.Sprintf(`
		Select %s.*
		FROM %s
		WHERE %s.deleted_at IS NULL
			AND %s = ?
		`, tableName, tableName, tableName, foreignKey)
		handleErrStr := fmt.Sprintf("\n\tif err != nil {\n\t\t return nil, err\n\t}")
		appendResultArray := fmt.Sprintf("\tresult := []domain.%s{}\n\tfor _, %s := range db%sRecords {\n\t\tresult = append(result, *%s.toEntity())\n\t}\n\n\treturn result, nil\n}", serviceName, tableName, serviceName, tableName)
		allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`, %s)\n%s\n\n%s\n\n", allQueryBlock, methodStr, methodContents, sqlQuery, foreignKey, handleErrStr, appendResultArray)
		byForeignKeyQueriesStr = byForeignKeyQueriesStr + allQueryBlock
	}

	return byForeignKeyQueriesStr
}

func ByForeignKeyAugmentedQueries(serviceName, tableName string, dbCols, varNames []string) string {
	var foreignKeyList []string
	var foreignKeyVarList []string
	var byForeignKeyAugmentedQueriesStr string
	var additionalTableStr string
	var joinStr string
	for i, dbCol := range dbCols {
		lenCol := len(dbCol)
		if dbCol[lenCol-3:lenCol] == "_id" {
			foreignKey := dbCol[len(tableName)+1:]
			foreignKeyList = append(foreignKeyList, foreignKey)
			foreignKeyVarList = append(foreignKeyVarList, varNames[i])
		}
		if dbCol[lenCol-3:lenCol] == "_id" { //TODO also make sure it doesn't contaim fromOU or to OU
			joinTable := dbCol[len(tableName)+1 : lenCol-3]
			additionalTableStr = additionalTableStr + ", " + joinTable + ".*"
			joinStr = fmt.Sprintf(`%s
		JOIN %s
			ON %s.%s_id = %s.id
			AND %s.deleted_at IS NULL`, joinStr, joinTable, tableName, joinTable, joinTable, joinTable)
		}
	}

	for i, foreignKey := range foreignKeyList {
		allQueryBlock := fmt.Sprintf("// By%sAugmented will retrieve all %s records in the database with a given %s.", foreignKeyVarList[i], serviceName, foreignKey)
		methodStr := fmt.Sprintf("func (s *%sService) By%sAugmented(%s string) ([]domain.%sAugmented, error) {", serviceName, foreignKeyVarList[i], foreignKey, serviceName)

		methodContents := fmt.Sprintf("\tdb%sRecords := []%sAugmented{}\n\terr := s.db.Connection().Select(&db%sRecords, `", serviceName, serviceName, serviceName)
		sqlQuery := fmt.Sprintf(`
		Select %s.*%s
		FROM %s%s
		WHERE %s.deleted_at IS NULL
			AND %s = ?
		`, tableName, additionalTableStr, tableName, joinStr, tableName, foreignKey)
		handleErrStr := fmt.Sprintf("\n\tif err != nil {\n\t\t return nil, err\n\t}")
		appendResultArray := fmt.Sprintf("\tresult := []domain.%sAugmented{}\n\tfor _, %s := range db%sRecords {\n\t\tresult = append(result, *%s.toEntityAugmented())\n\t}\n\n\treturn result, nil\n}", serviceName, tableName, serviceName, tableName)
		allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`,%s)\n%s\n\n%s\n\n", allQueryBlock, methodStr, methodContents, sqlQuery, foreignKey, handleErrStr, appendResultArray)

		byForeignKeyAugmentedQueriesStr = byForeignKeyAugmentedQueriesStr + allQueryBlock
	}
	return byForeignKeyAugmentedQueriesStr
}

func StoreQuery(serviceName, tableName string, dbCols, varNames []string) string {
	allQueryBlock := fmt.Sprintf("// Store will store a %s record in the database.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) Store(item *domain.%s) (*domain.%s, error) {", serviceName, serviceName, serviceName)
	methodContents := fmt.Sprint("\tres, err := s.db.Connection().Exec(`")
	fieldList := ""
	qList := ""
	vList := ""
	for i, dbCol := range dbCols {
		if varNames[i] != "ID" {
			if len(qList) == 0 {
				fieldList = dbCol[len(tableName)+1:]
				qList = "?"
				vList = "item." + varNames[i]
			} else {
				fieldList = fieldList + ", " + dbCol[len(tableName)+1:]
				qList = qList + ", ?"
				vList = vList + ", item." + varNames[i]
			}
		}
	}

	sqlQuery := fmt.Sprintf(`
		INSERT INTO %s
		(%s)
		VALUES
		(%s)
		`, tableName, fieldList, qList)

	handleReturnStr := fmt.Sprintf(`
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	itemCopy := *item
	itemCopy.ID = int(id)
	
	return &itemCopy, nil`)

	allQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`, %s)\n%s\n}", allQueryBlock, methodStr, methodContents, sqlQuery, vList, handleReturnStr)
	return allQueryBlock
}

func DeleteByIDQuery(serviceName, tableName string) string {
	deleteQueryBlock := fmt.Sprintf("// DeleteByID mark the %s record with the specified ID as deleted.", serviceName)
	methodStr := fmt.Sprintf("func (s *%sService) DeleteByID(id string) (error) {", serviceName)
	methodContents := fmt.Sprint("\t_, err := s.db.Connection().Exec(`")
	sqlQuery := fmt.Sprintf(`
		UPDATE %s
		SET deleted_at = NOW()
		WHERE id = ?
			AND deleted_at IS NULL
		LIMIT 1	
		`, tableName)
	handleReturnStr := fmt.Sprintf("\n\treturn err\n}")
	deleteQueryBlock = fmt.Sprintf("%s\n%s\n%s%s`, id)\n%s", deleteQueryBlock, methodStr, methodContents, sqlQuery, handleReturnStr)

	return deleteQueryBlock
}
