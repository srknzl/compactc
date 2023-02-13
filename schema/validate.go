package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"compactc/common"
	"github.com/xeipuuv/gojsonschema"
)

func ValidateWithJSONSchema(schema map[string]interface{}) (isValid bool, schemaErrors []string, err error) {
	jsonSchemaString, err := json.Marshal(schema)
	if err != nil {
		return false, nil, err
	}
	return validateJSONSchemaString(string(jsonSchemaString))
}

func ProcessSchema(schema common.Schema) error {
	var err error
	// From full name (packageName.className) to class
	classNames := make(map[string]common.Class)
	// process imports if exists
	if schema.Import != nil {
		for _, relativeYamlPath := range schema.Import {
			if err = processImport(relativeYamlPath, classNames); err != nil {
				return err
			}
		}
	}
	err = registerClassesAndCheckForDuplicate(schema, classNames)
	if err != nil {
		return err
	}
	err = checkAllFieldTypesAreValid(schema, classNames)
	if err != nil {
		return err
	}
	return nil
}

func checkAllFieldTypesAreValid(schema common.Schema, classNames map[string]common.Class) error {
	for _, c := range schema.Classes {
		fieldNames := make(map[string]struct{}, len(c.Fields))
		for _, f := range c.Fields {
			if _, ok := fieldNames[f.Name]; ok {
				return fmt.Errorf("validation error: '%s' field is defined more than once in class '%s'", f.Name, c.Name)
			}
			typ := f.Type
			fieldNames[f.Name] = struct{}{}
			if strings.HasSuffix(typ, "[]") {
				// if type is an array type, loose the brackets and validate underlying type
				typ = typ[:len(typ)-2]
			}
			if isBuiltInType(typ) {
				continue
			}
			// if field is external, we don't need to validate it
			if f.External {
				continue
			}
			// check if type is a class name
			if isCompactName(typ, classNames) {
				continue
			}
			return fmt.Errorf("validation error: field type '%s' is not one of the builtin types or not defined", typ)
		}
	}
	return nil
}

func registerClassesAndCheckForDuplicate(schema common.Schema, classNames map[string]common.Class) error {
	for _, cls := range schema.Classes {
		fullName, nameSpace := getClassFullNameAndNamespace(cls.Name, schema.Namespace)
		if _, ok := classNames[fullName]; ok {
			return fmt.Errorf("Class defined more than once. Compact class with name %s and namespace %s already exist", cls.Name, nameSpace)
		}
		cls.Namespace = nameSpace
		classNames[fullName] = cls
	}
	return nil
}

func getClassFullNameAndNamespace(className string, namespace string) (string, string) {
	if namespace == "" {
		return DefaultNamespace + "." + className, DefaultNamespace
	}
	return namespace + "." + className, namespace
}

func transcode(in, out interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return err
	}
	return json.NewDecoder(buf).Decode(out)
}

func validateJSONSchemaString(schema string) (isValid bool, schemaErrors []string, err error) {
	schemaLoader := gojsonschema.NewStringLoader(schema)
	vsl := gojsonschema.NewStringLoader(validationSchema)
	result, err := gojsonschema.Validate(vsl, schemaLoader)
	if err != nil {
		return false, nil, err
	}
	if isValid = result.Valid(); !isValid {
		for _, e := range result.Errors() {
			schemaErrors = append(schemaErrors, e.String())
		}
	}
	return isValid, schemaErrors, nil
}

func isBuiltInType(t string) bool {
	t = strings.ToLower(t)
	for _, bt := range builtinTypes {
		if t == strings.ToLower(bt) {
			return true
		}
	}
	return false
}

func isCompactName(typ string, compactNames map[string]common.Class) bool {
	for fullName := range compactNames {
		if typ == fullName {
			return true
		}
	}
	return false
}

var builtinTypes = []string{
	"boolean",
	"int8",
	"int16",
	"int32",
	"int64",
	"float32",
	"float64",
	"string",
	"decimal",
	"time",
	"date",
	"timestamp",
	"timestampWithTimezone",
	"nullableBoolean",
	"nullableInt8",
	"nullableInt16",
	"nullableInt32",
	"nullableInt64",
	"nullableFloat32",
	"nullableFloat64",
}

const validationSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "id": "https://github.com/hazelcast/hazelcast-client-protocol/blob/master/schema/protocol-schema.json",
  "title": "Hazelcast Client Protocol Definition",
  "type": "object",
  "definitions": {},
  "additionalProperties": false,
  "properties": {
    "namespace": {
      "type": "string"
    },
    "import": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "uniqueItems": true
    },
    "classes": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "name": {
            "type": "string"
          },
          "fields": {
            "type": "array",
            "items": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "name": {
                  "type": "string"
                },
                "type": {
                  "type": [
                    "string"
                  ]
                }
              },
              "required": [
                "name",
                "type"
              ]
            }
          }
        },
        "required": [
          "name",
          "fields"
        ]
      }
    }
  },
  "required": [
    "classes"
  ]
}`
