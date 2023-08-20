package markdown

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func GenerateMarkdown(file string, destination string) error {
	jsonFile, err := os.Open(file)
	if err != nil {
		log.Err(err).Msgf("fail to open file %s", file)
		return errors.New("fail to generate markdown")
	}

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Err(err).Msgf("fail to read file content %s", file)
		return errors.New("fail to generate markdown")
	}

	var schema = map[string]interface{}{}

	err = json.Unmarshal(byteValue, &schema)
	if err != nil {
		log.Err(err).Msgf("fail to unmarshal json file content %s ", file)
		return errors.New("fail to generate markdown")
	}

	content := Process(schema, "", filepath.Base(file))

	return writeFile(destination, content)
}

func writeFile(file string, content string) error {

	f, err := os.Create(file)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer f.Close()

	_, err = f.WriteString(content)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return nil
}

var coreSchemaTypes = []string{
	"array",
	"boolean",
	"integer",
	"number",
	"null",
	"object",
	"string",
}

func generateElementTitle(octothorpes string, elementName string, elementType string, isRequired bool, isEnum bool, example string) string {

	var text = []string{octothorpes}

	if elementName != "" {
		text = append(text, " `"+elementName+"`")
	}

	if elementType != "" || isRequired {
		text = append(text, " (")

		if elementType != "" {
			text = append(text, elementType)
		}

		if isEnum {
			text = append(text, ", enum")
		}

		if isRequired {
			text = append(text, ", required")
		}
		text = append(text, ")")
	}

	if example != "" {
		text = append(text, " eg: `"+example+"`")
	}

	return strings.Join(text, "")
}

func generateSinglePropertyRestriction(schema map[string]interface{}) func(string, string) string {
	return func(key string, text string) string {
		if _, exist := schema[key]; exist {
			var val = fmt.Sprintf("* %s : `%v`", text, schema[key])
			return val
		} else {
			return ""
		}
	}
}

func generatePropertyRestrictions(schema map[string]interface{}) string {

	var generate = generateSinglePropertyRestriction(schema)
	vals := []string{
		generate("minimum", "Minimum"),
		generate("maximum", "Maximum"),
		generate("pattern", "Regex pattern"),
		generate("minItems", "Minimum items"),
		generate("uniqueItems", "Unique items"),
	}

	return strings.Join(vals, "")
}

func getActualType(schema map[string]interface{}, subSchemas map[string]string) string {
	if typ, ok := schema["type"]; ok {
		if typeSr, ok := typ.(string); ok {
			return typeSr
		}
		if typesSlice, ok := typ.([]interface{}); ok {

			for _, val := range typesSlice {
				if strVal, ok := val.(string); ok {
					if !strings.EqualFold(strVal, "null") {
						return strVal
					}
				}
			}
		}

	} else if ref, refExist := schema["$ref"]; refExist {
		if val, exist := subSchemas[ref.(string)]; exist {
			return val
		}
		return fmt.Sprintf("%v", strings.TrimPrefix(ref.(string), "file://"))

	} else if _, oneOfExist := schema["oneOf"]; oneOfExist {

	}
	return ""
}

func isRequiredProperty(propertyKey string, schema map[string]interface{}) bool {

	if req, requiredExist := schema["required"]; requiredExist {
		if requiredFields, castOk := req.([]interface{}); castOk {
			for _, field := range requiredFields {
				if field == propertyKey {
					return true
				}
			}
		}
	}
	return false
}

// generatePropertiesTable generate the Markdown table summary of object properties
func generatePropertiesTable(octothorpes string, name string, schema map[string]interface{}, subSchemas map[string]string) string {

	var res []string

	if haveKey(schema, "properties") {

		res = append(res, fmt.Sprintf("%s %s Properties", octothorpes, name))

		res = append(res, fmt.Sprintf("|%s|", strings.Join([]string{"Property", "Type", "Required"}, "|")))
		res = append(res, fmt.Sprintf("|%s|", strings.Join([]string{":------", ":---", ":--------"}, "|")))

		for property, innerSchema := range schema["properties"].(map[string]interface{}) {

			actualType := getActualType(innerSchema.(map[string]interface{}), subSchemas)
			if actualType == "" {
				actualType = "-"
			}

			isRequired := isRequiredProperty(property, schema)

			res = append(res, fmt.Sprintf("|%s|%s|%v|",
				property,
				actualType,
				isRequired,
			))
		}

	}

	return strings.Join(res, "\n")
}

func generatePatternPropertySection(octothorpes string, schema map[string]interface{}, subSchemas map[string]string) []string {

	var res []string

	if patternProperties, ok := schema["patternProperties"]; ok {

		mapProperties := patternProperties.(map[string]interface{})

		for propertyKey, propertyContentVal := range mapProperties {

			var propertyContent = propertyContentVal.(map[string]interface{})
			var propertyIsRequired = isRequiredProperty(propertyKey, schema)
			var sections = generateSchemaSectionText(octothorpes+"#", propertyKey, propertyIsRequired, propertyContent, subSchemas)
			res = append(res, sections...)
		}
		return res
	}

	res = generateOneOf(schema, subSchemas)
	if res == nil {
		res = []string{}
	}

	return res
}

func generateOneOf(schema map[string]interface{}, subSchemas map[string]string) []string {

	var res []string

	if oneOfVal, exist := schema["oneOf"]; exist {

		var oneOf = oneOfVal.([]interface{})

		for _, innerSchema := range oneOf {
			actualType := getActualType(innerSchema.(map[string]interface{}), subSchemas)
			if actualType == "" {
				print(actualType)
			}
			res = append(res, fmt.Sprintf("* `%s`", actualType))
		}

		var oneOfList = strings.Join(res, "\n")

		return []string{"This property must be one of the following types:", oneOfList}
	}

	return res
}

func generatePropertySection(octothorpes string, schema map[string]interface{}, subSchemas map[string]string) []string {

	if properties, ok := schema["properties"]; ok {

		mapProperties := properties.(map[string]interface{})

		var res []string

		for propertyKey, propertyContentVal := range mapProperties {

			var propertyContent = propertyContentVal.(map[string]interface{})
			var propertyIsRequired = isRequiredProperty(propertyKey, schema)
			var sections = generateSchemaSectionText(octothorpes+"#", propertyKey, propertyIsRequired, propertyContent, subSchemas)
			res = append(res, sections...)
		}
		return res
	}

	var res = generateOneOf(schema, subSchemas)
	if res == nil {
		res = []string{}
	}

	return res

}

func getBoolVal(m map[string]interface{}, key string) bool {

	if m == nil {
		return false
	}

	val, ok := m[key]
	if !ok || val == nil {
		return false
	}

	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	return false
}

func getStringVal(m map[string]interface{}, key string) string {

	if m == nil {
		return ""
	}

	val, ok := m[key]
	if !ok || val == nil {
		return ""
	}
	return val.(string)
}

func haveKey(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	_, res := m[key]
	return res
}
func generateSchemaSectionText(octothorpes string, name string, isRequired bool, schema map[string]interface{}, subSchemas map[string]string) []string {

	var schemaType = getActualType(schema, subSchemas)

	var text = []string{

		generateElementTitle(octothorpes, name, schemaType, isRequired, haveKey(schema, "enum"), getStringVal(schema, "example")),
		getStringVal(schema, "description"),
	}

	if strings.EqualFold(schemaType, "object") {

		if _, ok := schema["properties"]; ok {

			tableLines := generatePropertiesTable(octothorpes, name, schema, subSchemas)
			text = append(text, tableLines)

			text = append(text, "Properties detail of the `"+name+"` object:")

			for _, section := range generatePropertySection(octothorpes, schema, subSchemas) {
				text = append(text, section)
			}
		}

	}

	if strings.EqualFold(schemaType, "array") {
		var haveItemsType = haveKey(schema, "items") && haveKey(schema["items"].(map[string]interface{}), "type")
		var itemsType = ""
		var items, _ = schema["items"].(map[string]interface{})

		if !haveItemsType && haveKey(items, "ref") {
			itemsType = getActualType(items, subSchemas)
		}

		if itemsType != "" && name != "" {
			text = append(text, "The object is an array with all elements of the type `"+itemsType+"`.")
		} else if itemsType != "" {
			text = append(text, "The schema defines an array with all elements of the type `"+itemsType+"`.")
		} else {

			var validationItems []any

			if haveKey(items, "allOf") {
				text = append(text, "The elements of the array must match *all* of the following properties:")
				validationItems = items["allOf"].([]any)
			} else if haveKey(items, "anyOf") {
				text = append(text, "The elements of the array must match *at least one* of the following properties:")
				validationItems = items["anyOf"].([]any)
			} else if haveKey(items, "oneOf") {
				text = append(text, "The elements of the array must match *exactly one* of the following properties:")
				validationItems = items["oneOf"].([]any)
			} else if haveKey(items, "not") {
				text = append(text, "The elements of the array must *not* match the following properties:")
				validationItems = items["not"].([]any)
			}

			if len(validationItems) > 0 {
				for _, itemVal := range validationItems {
					item := itemVal.(map[string]interface{})
					var title = getStringVal(item, "title")

					sections := generateSchemaSectionText(octothorpes, title, false, item, subSchemas)
					text = append(text, sections...)
				}
			}
		}
	}

	if haveKey(schema, "oneOf") {
		text = append(text, "The object must be one of the following types:")

		var oneOfTexts []string
		for _, oneVal := range schema["oneOf"].([]interface{}) {

			oneMap := oneVal.(map[string]interface{})

			if haveKey(oneMap, "$ref") {
				var ref = oneMap["$ref"].(string)
				oneOfTexts = append(oneOfTexts, fmt.Sprintf("* `%v`", subSchemas[ref]))
			} else {
				print("...")
			}

		}
		text = append(text, strings.Join(oneOfTexts, "\n"))
	}

	if haveKey(schema, "enum") {

		text = append(text, "This element must be one of the following enum values:")

		var enumsText []string
		for _, enumItem := range schema["enum"].([]interface{}) {
			enumsText = append(enumsText, fmt.Sprintf("* `%s`", enumItem))
		}

		text = append(text, strings.Join(enumsText, "\n"))
	}

	if defaultVal, ok := schema["default"]; ok && defaultVal != nil {

	}

	var restrictions = generatePropertyRestrictions(schema)

	if restrictions != "" {
		text = append(text, "Additional restrictions:")
		text = append(text, restrictions)
	}

	return text

}

func resolveDefinitionKey(schema map[string]interface{}) string {

	var defKey = "definitions"
	if haveKey(schema, "$def") {
		defKey = "$def"
	}
	return defKey
}

func Process(schema map[string]interface{}, startingOctothorpes string, filename string) string {

	var subSchemaTypes = make(map[string]string)

	var defKey = resolveDefinitionKey(schema)

	if def, exist := schema[defKey]; exist && def != nil {
		defMap, _ := def.(map[string]interface{})
		for key, _ := range defMap {
			subSchemaTypes["#/"+defKey+"/"+key] = key
		}
	}

	var text []string
	var octothorpes = startingOctothorpes

	text = append(text, octothorpes+"# "+filename)
	text = append(text, "\n", "---")

	if title, ok := schema["title"]; ok {
		octothorpes += "#"
		text = append(text, octothorpes+" "+title.(string))
	}

	if id, ok := schema["$id"]; ok {
		text = append(text, fmt.Sprintf("```text\n%s\n```", id.(string)))
	}

	if typ, _ := schema["type"]; typ == "object" {

		// Append description
		if desc, exist := schema["description"]; exist && desc != "" {
			text = append(text, desc.(string))
		}

		// Print properties
		sections := generatePropertySection(octothorpes, schema, subSchemaTypes)
		if len(sections) > 0 {
			text = append(text, "The schema defines the following properties:")
			for _, section := range generatePropertySection(octothorpes, schema, subSchemaTypes) {
				text = append(text, section)
			}
		}

	} else {
		text = append(text, generateSchemaSectionText("#"+octothorpes, "", false, schema, subSchemaTypes)...)
	}

	if defsVal, _ := schema[defKey]; defsVal != nil {
		text = append(text, "---")
		text = append(text, "# Sub Schemas")
		text = append(text, "The schema defines the following additional types:")

		var defs = defsVal.(map[string]interface{})

		for subSchemaTypeName, subSchemaDefVal := range defs {

			subSchemaDef := subSchemaDefVal.(map[string]interface{})

			subSchemaDefType := ""
			if _, ok := subSchemaDef["type"].(string); ok {
				subSchemaDefType = subSchemaDef["type"].(string)
			}

			text = append(text, "## `"+subSchemaTypeName+"` ("+subSchemaDefType+")")

			text = append(text, getStringVal(subSchemaDef, "description"))

			if typ, _ := subSchemaDef["type"]; typ == "object" {
				if propertiesVal, _ := subSchemaDef["properties"]; propertiesVal != nil {

					tableLines := generatePropertiesTable("###"+octothorpes, subSchemaTypeName, subSchemaDef, subSchemaTypes)
					text = append(text, tableLines)

				}
			}

			sections := generatePropertySection("##", subSchemaDef, subSchemaTypes)
			if len(sections) != 0 {
				text = append(text, fmt.Sprintf("%s %s properties detail", octothorpes+"###", subSchemaTypeName))
				text = append(text, "Properties detail of the `"+subSchemaTypeName+"` object:")
				text = append(text, sections...)
			}

			sections = generatePatternPropertySection("##", subSchemaDef, subSchemaTypes)
			if len(sections) != 0 {
				text = append(text, fmt.Sprintf("%s %s patternProperties detail", octothorpes+"###", subSchemaTypeName))
				text = append(text, "PatternProperties detail of the `"+subSchemaTypeName+"` object:")
				text = append(text, sections...)
			}

		}
	}

	return strings.Join(text, "\n\n")
}
