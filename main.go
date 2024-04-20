package main

import (
	"fmt"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/parser"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
)

type QueryContext map[string]ast.Element

type ElementGroup []ElementWithContext
type ElementWithContext struct {
	context QueryContext
	element ast.Element
}

func (g ElementGroup) Do(f func(QueryContext)) {
	for _, element := range g {
		f(element.context)
	}
}

func (g ElementGroup) Where(f func(QueryContext) bool) ElementGroup {
	var elements ElementGroup
	for _, element := range g {
		if f(element.context) {
			elements = append(elements, element)
		}
	}
	return elements
}

func (g ElementGroup) Each(variable string, elementTypes []ast.ElementType) ElementGroup {
	var elements ElementGroup

	for _, elementWithContext := range g {
		var walker func(ast.Element)
		walker = func(e ast.Element) {
			if slices.Contains(elementTypes, e.ElementType()) {
				ec := ElementWithContext{element: e, context: QueryContext{}}
				for k, v := range elementWithContext.context {
					ec.context[k] = v
				}
				ec.context[variable] = e
				elements = append(elements, ec)
			}
			e.Walk(walker)
		}
		elementWithContext.element.Walk(walker)
	}

	return elements
}

var elementTypes map[string][]ast.ElementType
var elementTypesHelp map[string]bool

func getElementType(t string) []ast.ElementType {
	if elementTypes == nil {
		elementTypes = make(map[string][]ast.ElementType)
		elementTypesHelp = make(map[string]bool)

		expressions := []ast.ElementType{}
		statements := []ast.ElementType{}
		declarations := []ast.ElementType{}

		for i := 1; i < 50; i++ {
			elementType := strings.Replace(ast.ElementType(i).String(), "ElementType", "", 1)
			elementTypes[strings.ToLower(elementType)] = []ast.ElementType{ast.ElementType(i)}

			//try short name
			ty := elementType

			if strings.HasSuffix(ty, "Declaration") {
				declarations = append(declarations, ast.ElementType(i))
				ty = strings.Replace(ty, "Declaration", "", 1)
			}
			if strings.HasSuffix(ty, "Expression") {
				expressions = append(expressions, ast.ElementType(i))
				ty = strings.Replace(ty, "Expression", "", 1)
			}
			if strings.HasSuffix(ty, "Statement") {
				statements = append(statements, ast.ElementType(i))
				ty = strings.Replace(ty, "Statement", "", 1)
			}

			if _, ok := elementTypes[strings.ToLower(ty)]; !ok {
				elementTypes[strings.ToLower(ty)] = []ast.ElementType{ast.ElementType(i)}
				//add short to help
				elementTypesHelp[ty] = true
			} else {
				//can't add comparator, add long to help
				elementTypesHelp[elementType] = true
			}
		}

		//add groups
		elementTypesHelp["Statement"] = true
		elementTypes["statement"] = statements

		elementTypesHelp["Declaration"] = true
		elementTypes["declaration"] = declarations

		elementTypesHelp["Expression"] = true
		//todo: not sure about this, check
		elementTypes["expression"] = append(expressions, ast.ElementTypeExpressionStatement)

	}

	if v, ok := elementTypes[strings.ToLower(t)]; ok {
		return v
	}
	return nil
}

func reflectGetField(typeName string, value ast.Element, fieldName string, isOptional bool) reflect.Value {

	v := reflect.Indirect(reflect.ValueOf(value))

	fieldValue := v.FieldByName(fieldName)

	if !fieldValue.IsValid() {
		if isOptional {
			return fieldValue
		}
		keys := make([]string, v.Type().NumField())
		for i := 0; i < v.NumField(); i++ {
			keys[i] = v.Type().Field(i).Name
		}
		sort.Strings(keys)
		panic(
			fmt.Sprintf("Invalid field `%s` on variable `%s`.\nAvailable fields: %s",
				fieldName,
				typeName,
				strings.Join(keys, ", "),
			),
		)
	}

	return fieldValue
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()

	if len(os.Args) != 3 {
		panic("cdcq <cadence file> <filter>")
	}
	filePath := os.Args[1]
	queryParts := strings.Split(os.Args[2], "|")

	if len(queryParts) != 2 {
		panic("Query should be in the format of: `<search> | <display>` )")
	}

	query := queryParts[0]
	display := queryParts[1]

	code, _ := os.ReadFile(filePath)
	program, err := parser.ParseProgram(nil, code, parser.Config{})
	if err != nil {
		return
	}

	root := ElementGroup{ElementWithContext{QueryContext{}, program}}

	c := 0

	typeName := ""
	//parse query
	for c < len(query) {
		var op = query[c]
		switch op {
		case ' ':
			c = c + 1
			break

		case '.': //each
			var t strings.Builder
			c = c + 1
			for {
				if query[c] == '[' || query[c] == '.' || query[c] == ' ' {
					break
				}
				t.Write([]byte{query[c]})
				c = c + 1
			}
			typeName = t.String()
			elementTypesToFilter := getElementType(typeName)
			if elementTypesToFilter == nil {
				keys := make([]string, len(elementTypesHelp))
				i := 0
				for k := range elementTypesHelp {
					keys[i] = k
					i = i + 1
				}
				sort.Strings(keys)
				panic(
					fmt.Sprintf("Invalid element type `%s`.\nAvailable types: %s",
						typeName,
						strings.Join(keys, ", "),
					),
				)
			}
			root = root.Each(typeName, elementTypesToFilter)
			break
		case '[': //where

			var t strings.Builder
			c = c + 1
			for {
				if query[c] == ']' || query[c] == '=' {
					break
				}
				t.Write([]byte{query[c]})
				c = c + 1
			}
			field := t.String()

			t.Reset()
			c = c + 1
			for {
				if query[c] == ']' {
					c = c + 1
					break
				}
				t.Write([]byte{query[c]})
				c = c + 1
			}
			value := t.String()

			root = root.Where(func(context QueryContext) bool {
				isNot := false
				fieldName := field
				lookupValue := value
				var comparator = strings.EqualFold

				if strings.HasPrefix(value, "~") {
					comparator = strings.Contains
					lookupValue = lookupValue[1:]
				}

				if strings.HasPrefix(value, "'") {
					lookupValue = lookupValue[1 : len(lookupValue)-1]
				}

				if strings.HasSuffix(field, "!") {
					isNot = true
					fieldName = fieldName[:len(fieldName)-1]
				}

				element, ok := context[typeName]
				if !ok {
					panic("invalid query")
				}

				fieldValue := reflectGetField(typeName, element, fieldName, false)

				data := fmt.Sprintf("%s", fieldValue)
				if fieldValue.Kind() == reflect.Slice {
					data = data[1 : len(data)-1]
				}
				b := comparator(
					strings.ToLower(data),
					strings.ToLower(lookupValue),
				)

				if isNot {
					return !b
				}
				return b

			})
			break
		}
	}

	var variables []string
	// extract variables from display
	c = 0
	for c < len(display) {
		var op = display[c]
		switch op {
		case '{':
			c = c + 1
			variableName := ""
			for {
				if display[c] == '}' {
					c = c + 1
					break
				}
				variableName = fmt.Sprintf("%s%c", variableName, display[c])
				c = c + 1
			}
			variables = append(variables, variableName)
		default:
			c = c + 1
			break
		}
	}

	root.Do(func(context QueryContext) {
		result := display
		for _, v := range variables {
			parts := strings.Split(v, ".")
			object := parts[0]
			field := parts[1]

			isOptional := false
			if strings.HasSuffix(object, "?") {
				isOptional = true
				object = object[:len(object)-1]
			}
			data := ""
			if v == "Path" {
				data = filePath
			} else {
				vv, ok := context[object]
				if !ok {
					keys := make([]string, len(context))
					i := 0
					for k := range context {
						keys[i] = k
						i++
					}
					sort.Strings(keys)
					panic(fmt.Sprintf("invalid variable `%s`. Available variables: %s", object, strings.Join(keys, ", ")))
				}

				t := reflect.Indirect(reflect.ValueOf(vv).Elem())
				if len(parts) == 1 {
					data = fmt.Sprintf("%s", vv)
				} else {
					t = reflect.ValueOf(vv)
					m := t.MethodByName(field)
					if m.IsValid() {
						data = m.Call([]reflect.Value{})[0].String()
					} else {
						v := reflectGetField(object, vv, field, isOptional)
						data = fmt.Sprintf("%s", v)
						data = strings.ReplaceAll(data, "<invalid reflect.Value>", "<invalid field>")

					}
				}
			}
			result = strings.ReplaceAll(result, fmt.Sprintf("{%s}", v), data)
		}
		fmt.Println(result)
	})
}
