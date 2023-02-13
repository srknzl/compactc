package schema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONSchemaString(t *testing.T) {
	tcs := []struct {
		name        string
		schema      string
		isErr       bool
		noSchemaErr bool
		errString   string
	}{
		{
			name:   "non-json string",
			schema: "",
			isErr:  true,
		},
		{
			name:        "valid",
			schema:      Valid,
			noSchemaErr: true,
		},
		{
			name:        "valid custom field type defined in schema",
			schema:      ValidNewFieldTypeDefined,
			noSchemaErr: true,
		},
		{
			name:      "mandatory class field is missing",
			schema:    "{}",
			errString: "classes is required",
		},
		{
			name: "mandatory 'fields' field of class is missing",
			schema: `{ "classes":[
                     {
                        "name":"Employee"
                     }
                  ]
           }`,
			errString: "fields is required",
		},
		{
			name: "mandatory 'name' field in 'fields' field is missing",
			schema: `{ "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "type":"Work"
                           }
                        ]
                     }
                  ]
           }`,
			errString: "name is required",
		},
		{
			name: "mandatory 'type' field in 'fields' field is missing",
			schema: `{ "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age"
                           }
                        ]
                     }
                  ]
           }`,
			errString: "type is required",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			valid, errors, err := validateJSONSchemaString(tc.schema)
			assert.Equal(t, tc.isErr, err != nil)
			if tc.isErr {
				return
			}
			assert.Nil(t, err)
			if tc.noSchemaErr {
				assert.Empty(t, errors)
				assert.True(t, valid)
				return
			}
			assert.False(t, valid)
			assert.Contains(t, strings.Join(errors, ","), tc.errString)
		})
	}
}

func TestValidateSchemaSemantics(t *testing.T) {
	tcs := []struct {
		name   string
		schema string
		isErr  bool
		err    string
	}{
		{
			name:   "invalid field type",
			schema: InvalidFieldType,
			isErr:  true,
			err:    "not one of the builtin types or not defined",
		},
		{
			name:   "valid field type defined in schema",
			schema: ValidNewFieldTypeDefined,
		},
		{
			name: "duplicate compact class",
			schema: `{
                  "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age",
                              "type":"Work"
                           }
                        ]
                     },
                     {
                        "name":"Employee",
                        "fields":[]
                     }
                  ]
           }`,
			isErr: true,
			err:   "already exist",
		},
		{
			name: "duplicate field name same field type",
			schema: `{
                  "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age",
                              "type":"int8"
                           },
                           {
                              "name":"age",
                              "type":"int8"
                           }
                        ]
                     }
                  ]
           }`,
			isErr: true,
			err:   "field is defined more than once in class",
		},
		{
			name: "duplicate field name different field type",
			schema: `{
                  "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age",
                              "type":"int8"
                           },
                           {
                              "name":"age",
                              "type":"string"
                           }
                        ]
                     }
                  ]
           }`,
			isErr: true,
			err:   "field is defined more than once in class",
		},
		{
			name: "valid array field type",
			schema: `{
                  "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age",
                              "type":"nullableInt16[]"
                           }
                        ]
                     }
                  ]
           }`,
		},
		{
			name: "invalid array field type",
			schema: `{
                  "classes":[
                     {
                        "name":"Employee",
                        "fields":[
                           {
                              "name":"age",
                              "type":"[]"
                           }
                        ]
                     }
                  ]
           }`,
			isErr: true,
			err:   "not one of the builtin types or not defined",
		},
		{
			name:   "can import another yaml file and use its class",
			schema: ImportedFieldType,
		},
		{
			name:   "can import multiple other yaml files and use types with the same class name",
			schema: MultipleImportedFieldTypeWithSameClassName,
		},
		{
			name:   "Defining a class as external makes it usable even when not imported",
			schema: ExternalFieldType,
		},
		{
			name:   "defining and importing the same class causes error",
			schema: SameClassImportedError,
			isErr:  true,
			err:    "Class defined more than once",
		},
		{
			name:   "importing and defining the same named and namespaced class causes error",
			schema: SameNamespaceAndNamedClassImportedError,
			isErr:  true,
			err:    "Class defined more than once",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var schema map[string]interface{}
			require.Nil(t, json.Unmarshal([]byte(tc.schema), &schema))
			sch, err := ConvertMapToSchema(schema)
			require.Nil(t, err)
			err = ProcessSchema(sch)
			if !tc.isErr {
				assert.Nil(t, err)
				return
			}
			if err == nil {
				t.Fatal("expected error but got none")
			}
			assert.Contains(t, err.Error(), tc.err)
		})
	}
}

const (
	Valid = `{ 
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"age",
                  "type":"int32[]"
               },
               {
                  "name":"name",
                  "type":"string"
               },
               {
                  "name":"id",
                  "type":"int64"
               }
            ]
         }
      ]
   }`
	// Field is not defined nor imported nor external
	InvalidFieldType = `{
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"age",
                  "type":"Work"
               }
            ]
         }
      ]
   }`
	// Adding external makes this work
	ExternalFieldType = `{
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"age",
                  "type":"Work",
                  "external":true
               }
            ]
         }
      ]
   }`
	ValidNewFieldTypeDefined = `{
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"age",
                  "type":"com.example.Work"
               }
            ]
         },
         {
            "name":"Work",
            "fields":[]
         }
      ]
   }`
	ImportedFieldType = `{
      "import":["xyz.yaml"], 
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"age",
                  "type":"com.xyz.Work"
               }
            ]
         }
      ]
   }`
	MultipleImportedFieldTypeWithSameClassName = `{
      "import":["xyz.yaml", "zyx.yaml"], 
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"work",
                  "type":"com.xyz.Work"
               },
               {
                  "name":"work2",
                  "type":"com.zyx.Work"
               }
            ]
         }
      ]
   }`
	// In this, imported class and defined class are exactly the same.
	SameClassImportedError = `{
      "import":["example.yml"], 
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"name",
                  "type":"string"
               }
            ]
         }
      ]
   }`

	// In this, the type is different.
	SameNamespaceAndNamedClassImportedError = `{
      "import":["example.yml"], 
      "classes":[
         {
            "name":"Employee",
            "fields":[
               {
                  "name":"work",
                  "type":"string"
               }
            ]
         }
      ]
   }`
)
