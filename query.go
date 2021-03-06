package GABAORM

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type QueryInf struct {
	TableName string
	Columns string
	Values string
}

func getExistingQuery(tableName string,queries *[]QueryInf) *QueryInf{
	for _ , query := range *queries{
		if query.TableName == tableName{
			return &query
		}
	}
	return nil
}

func AssignFromArgs(model interface{},columns *[]interface{})  {
	valueinf := reflect.ValueOf(model).Elem()
	typeinf := valueinf.Type()
	for i := 0; i < valueinf.NumField(); i++ {
		fieldkind := typeinf.Field(i).Type.Kind()
		if fieldkind==reflect.Slice || (fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") != "") {
			continue
		}
		if typeinf.Field(i).Type.String() == "time.Time" {
			*columns = append(*columns,valueinf.Field(i).Addr().Interface())
		}else if fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") == "" {
			AssignFromArgs(valueinf.FieldByName(typeinf.Field(i).Name).Addr().Interface(),columns)
		} else {
			*columns = append(*columns,valueinf.Field(i).Addr().Interface())
		}
	}
}

func extractColumnsFromStructWithCount(model interface{},column *string) {
	typeinf := reflect.TypeOf(model)
	valueinf := reflect.ValueOf(model)
	for i := 0; i < typeinf.NumField(); i++ {
		fieldkind := typeinf.Field(i).Type.Kind()
		if fieldkind == reflect.Slice || (fieldkind == reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") != "") {
			continue
		}
		if typeinf.Field(i).Type.String() == "time.Time" {
			if *column != "" {
				*column += ","
			}
			*column += getColumnName(typeinf.Field(i).Name)
		} else if fieldkind == reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") == "" {
			extractColumnsFromStructWithCount(valueinf.FieldByName(typeinf.Field(i).Name).Interface(), column)
		} else {
			if *column != "" {
				*column += ","
			}
			*column += getColumnName(typeinf.Field(i).Name)
		}
	}
}


func extractColumnsFromStruct(model interface{},column *string,currentTime time.Time)  {
	typeinf := reflect.TypeOf(model)
	valueinf := reflect.ValueOf(model)
	for i := 0; i < typeinf.NumField(); i++ {
		fieldkind := typeinf.Field(i).Type.Kind()
		if fieldkind==reflect.Slice || (fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") != "") {
			continue
		}
		if typeinf.Field(i).Type.String() == "time.Time" {
			if currentTime.IsZero(){
				continue
			}
			if *column != ""{
				*column += ","
			}
			*column += getColumnName(typeinf.Field(i).Name)
		}else if fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") == "" {
			extractColumnsFromStruct(valueinf.FieldByName(typeinf.Field(i).Name).Interface(),column,currentTime)
		} else {
			if *column != ""{
				*column += ","
			}
			*column += getColumnName(typeinf.Field(i).Name)
		}
	}
}

func extractValuesFromStruct(model interface{},value *string,isUpdate bool,queries *[]QueryInf,currentTime time.Time,primaryKeys map[string]interface{})  {
	typeinf := reflect.TypeOf(model)
	valueinf := reflect.ValueOf(model)
	for i := 0; i < typeinf.NumField(); i++ {
		fieldkind := typeinf.Field(i).Type.Kind()
		if (fieldkind==reflect.Slice || fieldkind == reflect.Array) && typeinf.Field(i).Tag.Get("foreignKey") == "" {
			continue
		} else if fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("foreignKey") != ""{
			foreignKeyName := typeinf.Field(i).Tag.Get("foreignKey")
			foreignChild := valueinf.Field(i).Interface()
			//TODO: References??????????????????
			reflect.ValueOf(valueinf.Field(i).FieldByName(foreignKeyName)).Elem().Set(reflect.ValueOf(primaryKeys["ID"]))
			modifyQuery(foreignChild,isUpdate,queries,currentTime)
			continue
		} else if (fieldkind==reflect.Slice || fieldkind == reflect.Array) && typeinf.Field(i).Tag.Get("foreignKey") != ""{
			if valueinf.Field(i).Len() > 0{
				bulkModifyQuery(valueinf.Field(i).Interface(),isUpdate,queries,currentTime)
			}
			continue
		} else if typeinf.Field(i).Tag.Get("gabaorm") != ""{
			if strings.Contains(typeinf.Field(i).Tag.Get("gabaorm"), "primarykey"){
				primaryKeys[typeinf.Field(i).Name] = valueinf.Field(i).Interface()
			}
		}
		if typeinf.Field(i).Type.String() == "time.Time" {
			if currentTime.IsZero(){
				continue
			}
			datetime := valueinf.Field(i).Interface().(time.Time)
			if typeinf.Field(i).Name=="CreatedAt" {
				datetime = currentTime
			}else if typeinf.Field(i).Name=="UpdatedAt" {
				datetime = currentTime
			}
			if *value != ""{
				*value += ","
			}
			datestring := datetime.Format(layout)
			if datetime.IsZero(){
				*value += "null"
			}else{
				*value += "'"+datestring+"'"
			}

		}else if fieldkind==reflect.Struct && typeinf.Field(i).Tag.Get("ForeignKey") == "" {
			newPrimaryKeys := map[string]interface{}{}
			extractValuesFromStruct(valueinf.FieldByName(typeinf.Field(i).Name).Interface(),value,isUpdate,queries,currentTime,newPrimaryKeys)
		} else {
			fmt.Println(typeinf.Field(i).Name)
			fmt.Println(valueinf.FieldByName(typeinf.Field(i).Name))
			if *value != ""{
				*value += ","
			}
			if fieldkind == reflect.String{
				*value += "'"+valueinf.Field(i).String()+"'"
			}else{
				*value += fmt.Sprintf("%v",valueinf.Field(i))
			}
		}
	}
}

func modifyQuery(model interface{},isUpdate bool,queries *[]QueryInf,currentTime time.Time){
	typeinf := reflect.TypeOf(model)
	tableName := getTableName(typeinf.Name())
	var queryInf *QueryInf
	existingQuery := getExistingQuery(tableName,queries)
	isNew := false
	if existingQuery != nil{
		queryInf = existingQuery
	}else{
		queryInf = &QueryInf{}
		queryInf.TableName = tableName
		isNew = true
	}
	primaryKeys := map[string]interface{}{}
	extractColumnsFromStruct(model,&queryInf.Columns,currentTime)
	extractValuesFromStruct(model,&queryInf.Values,isUpdate,queries,currentTime,primaryKeys)
	queryInf.Values = "("+queryInf.Values+")"
	if isNew{
		*queries = append(*queries, *queryInf)
	}
}

func bulkModifyQuery(models interface{},isUpdate bool,queries *[]QueryInf,currentTime time.Time){
	objects := reflect.ValueOf(models)
	typeinf := reflect.TypeOf(objects.Index(0).Interface())
	tableName := getTableName(typeinf.Name())
	var queryInf *QueryInf
	existingQuery := getExistingQuery(tableName,queries)
	isNew := false
	if existingQuery != nil{
		queryInf = existingQuery
	}else{
		queryInf = &QueryInf{}
		queryInf.TableName = tableName
		isNew = true
	}
	extractColumnsFromStruct(objects.Index(0).Interface(),&queryInf.Columns,currentTime)
	for i := 0; i < objects.Len(); i++{
		if queryInf.Values != ""{
			queryInf.Values += ","
		}
		childvalue := ""
		primaryKeys := map[string]interface{}{}
		extractValuesFromStruct(objects.Index(i).Interface(),&childvalue,isUpdate,queries,currentTime,primaryKeys)
		queryInf.Values += "("+childvalue+")"
	}
	if isNew{
		*queries = append(*queries, *queryInf)
	}
}

