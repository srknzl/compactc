package schema

import (
	"compactc/common"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

const DefaultNamespace = "com.example"

func ParseSchemaText(schema string) (common.Schema, error) {
	schemaMap, err := YAMLToMap([]byte(schema))
	if err != nil {
		return common.Schema{}, err
	}
	isValid, schemaErrors, err := ValidateWithJSONSchema(schemaMap)
	if err != nil {
		return common.Schema{}, err
	}
	if !isValid {
		return common.Schema{}, fmt.Errorf("Schema is not valid, validation errors:\n%s\n", strings.Join(schemaErrors, "\n"))
	}
	sch, err := ConvertMapToSchema(schemaMap)
	err = ProcessSchema(sch)
	if err != nil {
		return common.Schema{}, err
	}
	return sch, nil
}

func ConvertMapToSchema(schemaMap map[string]interface{}) (common.Schema, error) {
	var schema common.Schema
	if err := transcode(schemaMap, &schema); err != nil {
		return common.Schema{}, err
	}
	return schema, nil
}

func YAMLToMap(yamlSchema []byte) (map[string]interface{}, error) {
	s := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(yamlSchema, &s); err != nil {
		return nil, err
	}
	// convert map[interface{}]interface{} to map[string]interface{}
	i := ConvertMapI2MapS(s)
	schema, ok := i.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("malformed schema")
	}
	return schema, nil
}

func processImport(relativeYamlPath string, classNames map[string]common.Class) error {
	// read the yaml file
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	yamlSchema, err := os.ReadFile(filepath.Join(pwd, relativeYamlPath))
	if err != nil {
		return err
	}
	sch, err := ParseSchemaText(string(yamlSchema))
	if err != nil {
		return err
	}
	if sch.Import != nil {
		for _, relYamlPath := range sch.Import {
			if err = processImport(relYamlPath, classNames); err != nil {
				return err
			}
		}
	}
	err = registerClassesAndCheckForDuplicate(sch, classNames)
	if err != nil {
		return err
	}
	err = checkAllFieldTypesAreValid(sch, classNames)
	if err != nil {
		return err
	}
	return nil
}
