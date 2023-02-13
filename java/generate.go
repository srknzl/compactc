package java

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"compactc/common"
)

var javaTypes = map[string]string{
	"boolean":               "boolean",
	"int8":                  "byte",
	"int16":                 "short",
	"int32":                 "int",
	"int64":                 "long",
	"float32":               "float",
	"float64":               "double",
	"string":                "java.lang.String",
	"decimal":               "java.math.BigDecimal",
	"time":                  "java.time.LocalTime",
	"date":                  "java.time.LocalDate",
	"timestamp":             "java.time.LocalDateTime",
	"timestampWithTimezone": "java.time.OffsetDateTime",
	"nullableBoolean":       "Boolean",
	"nullableInt8":          "Byte",
	"nullableInt16":         "Short",
	"nullableInt32":         "Integer",
	"nullableInt64":         "Long",
	"nullableFloat32":       "Float",
	"nullableFloat64":       "Double",
}

type TypeInfo struct {
	Type          string
	IsCustomClass bool
	IsArr         bool
	FullType      string
}

func arrayOf(fieldType string) string {
	if strings.HasSuffix(fieldType, "[]") {
		return fieldType[0 : len(fieldType)-2]
	}
	return fieldType
}

func methodName(field TypeInfo, compactType string) string {
	var mn string
	if field.IsCustomClass {
		mn = "compact"
	} else {
		mn = compactType
	}
	// capitalize fist letter
	mn = strings.ToUpper(string(mn[0])) + mn[1:]
	if field.IsArr {
		return fmt.Sprintf("ArrayOf%s", arrayOf(mn))
	}
	return mn
}

func toJavaType(fieldType string) TypeInfo {
	var ti TypeInfo
	var ok bool
	if strings.HasSuffix(fieldType, "[]") {
		base := arrayOf(fieldType)
		ti.IsArr = true
		ti.Type, ok = javaTypes[base]
		if ok {
			ti.FullType = ti.Type + "[]"
		} else {
			ti.IsCustomClass = true
			ti.Type = base
			ti.FullType = fieldType
		}
	} else {
		ti.Type, ok = javaTypes[fieldType]
		if !ok {
			ti.Type = fieldType
			ti.IsCustomClass = true
		}
		ti.FullType = ti.Type
	}
	return ti
}

func fieldTypeAndNames(cls common.Class) string {
	var sb strings.Builder
	for _, f := range cls.Fields {
		ti := toJavaType(f.Type)
		sb.WriteString(fmt.Sprintf("%s %s, ", ti.FullType, f.Name))
	}
	s := sb.String()
	if len(s) >= 2 {
		s = s[:len(s)-2]
	}
	return s
}

func fieldNames(class common.Class) string {
	var sb strings.Builder
	for _, f := range class.Fields {
		sb.WriteString(fmt.Sprintf("%s, ", f.Name))
	}
	content := sb.String()
	return content[:len(content)-2]
}

func hashcodeBody(cls common.Class) string {
	const indentation = "        "
	var content string
	var isTempDeclared bool
	for _, field := range cls.Fields {
		var line string
		fn := field.Name
		switch field.Type {
		case "boolean":
			line = fmt.Sprintf("result = 31 * result + (%s ? 1 : 0);", fn)
		case "int64":
			line = fmt.Sprintf("result = 31 * result + (int) (%s ^ (%s >>> 32));", fn, fn)
		case "float32":
			line = fmt.Sprintf("result = 31 * result + (%s != +0.0f ? Float.floatToIntBits(%s) : 0);", fn, fn)
		case "float64":
			if !isTempDeclared {
				line = "long temp;\n"
				isTempDeclared = true
			}
			line += fmt.Sprintf(`%stemp = Double.doubleToLongBits(%s);
%sresult = 31 * result + (int) (temp ^ (temp >>> 32));`, indentation, fn, indentation)
		default:
			if strings.HasSuffix(field.Type, "[]") {
				line = fmt.Sprintf("result = 31 * result + Arrays.hashCode(%s);", field.Name)
			} else if _, ok := common.FixedSizeTypes[field.Type]; ok {
				line = fmt.Sprintf("result = 31 * result + (int) %s;", field.Name)
			} else {
				line = fmt.Sprintf("result = 31 * result + Objects.hashCode(%s);", field.Name)
			}
		}

		content += "        " + line + "\n"
	}
	return content
}

func toStringBody(field common.Field) string {
	if strings.HasSuffix(field.Type, "[]") {
		return fmt.Sprintf("                + \", + %s=\" + Arrays.toString(%s)\n", field.Name, field.Name)
	}
	return fmt.Sprintf(`                + ", + %s=" + %s
`, field.Name, field.Name)
}

func fields(field common.Field) string {
	ti := toJavaType(field.Type)
	return fmt.Sprintf("    private %s %s;", ti.FullType, field.Name)
}

func read(field common.Field) string {
	ti := toJavaType(field.Type)
	if ti.IsArr && ti.IsCustomClass {
		return fmt.Sprintf(`            %s %s = reader.readArrayOfCompact("%s", %s.class);`, ti.FullType, field.Name, field.Name, ti.Type)
	}
	return fmt.Sprintf(`            %s %s = reader.read%s("%s");`, ti.FullType, field.Name, methodName(ti, field.Type), field.Name)
}

func getters(field common.Field) string {
	ti := toJavaType(field.Type)
	upperName := strings.ToUpper(string(field.Name[0])) + field.Name[1:]
	return fmt.Sprintf(`    public %s get%s() {
        return %s;
    }
`, ti.FullType, upperName, field.Name)
}

func Generate(schema common.Schema) map[string]string {
	classes := make(map[string]string)
	for _, cls := range schema.Classes {
		var sb strings.Builder
		err := GenerateClass(cls, &sb)
		if err != nil {
			fmt.Println(err)
		}
		classes[cls.Name] = sb.String()
	}
	return classes
}

func GenerateClass(cls common.Class, w io.Writer) error {
	tmpl, err := template.New("main").Funcs(template.FuncMap{
		"read":              read,
		"toJavaType":        toJavaType,
		"methodName":        methodName,
		"fields":            fields,
		"fieldTypeAndNames": fieldTypeAndNames,
		"getters":           getters,
		"equalsBody":        equalsBody,
		"hashcodeBody":      hashcodeBody,
		"toStringBody":      toStringBody,
		"fieldNames":        fieldNames,
	}).Parse(bodyTemplate)
	for _, t := range []struct {
		templateName string
		template     string
	}{
		{
			"compactSerDeser",
			compactSerDeserTemplate,
		},
		{
			"imports",
			importsTemplate,
		},
		{
			"constructors",
			constructorsTemplate,
		},
	} {
		tmpl, err := tmpl.New(t.templateName).Parse(t.template)
		tmpl = template.Must(tmpl, err)
	}
	err = tmpl.Execute(w, struct {
		common.Class
	}{
		cls,
	})
	return err
}

func equalsBody(field common.Field) string {
	ti := toJavaType(field.Type)
	var s string
	if field.Type == "float32" {
		s = fmt.Sprintf("if (Float.compare(%s, that.%s) != 0) return false;", field.Name, field.Name)
	} else if field.Type == "float64" {
		s = fmt.Sprintf("if (Double.compare(%s, that.%s) != 0) return false;", field.Name, field.Name)
	} else if ti.IsArr {
		s = fmt.Sprintf("if (!Arrays.equals(%s, that.%s)) return false;", field.Name, field.Name)
	} else if _, ok := common.FixedSizeTypes[field.Type]; ok {
		s = fmt.Sprintf("if (%s != that.%s) return false;", field.Name, field.Name)
	} else {
		s = fmt.Sprintf("if (!Objects.equals(%s, that.%s)) return false;", field.Name, field.Name)
	}
	return "        " + s
}

const importsTemplate = `package {{.Namespace}};

import com.hazelcast.nio.serialization.compact.CompactReader;
import com.hazelcast.nio.serialization.compact.CompactSerializer;
import com.hazelcast.nio.serialization.compact.CompactWriter;

import javax.annotation.Nonnull;
import java.util.Arrays;
import java.util.Objects;`

const compactSerDeserTemplate = `public static final class Serializer implements CompactSerializer<{{.Name}}> {
        @Nonnull
        @Override
        public {{ .Name }} read(@Nonnull CompactReader reader) {

{{range $field := .Fields}}{{read $field}}
{{end}}            return new {{ .Name }}({{ fieldNames .Class }});
        }

        @Override
        public void write(@Nonnull CompactWriter writer, @Nonnull {{ .Name }} object) {
{{range $field := .Fields}}
            writer.write{{methodName (toJavaType $field.Type) $field.Type }}("{{ $field.Name }}", object.{{ $field.Name }});{{end}}
        }
    };

    public static final CompactSerializer<{{ .Name }}> HZ_COMPACT_SERIALIZER = new Serializer();`

const constructorsTemplate = `public {{ .Name }}() {
    }

    public {{ .Name }}({{fieldTypeAndNames .Class}}) {
{{range $field := .Fields}}
        this.{{$field.Name}} = {{$field.Name}};{{end}}
    }`

const bodyTemplate = `{{ template "imports" $}}

public class {{ .Name }} {

    {{ template "compactSerDeser" $}}

{{range $field := .Fields}}{{fields $field}}
{{end}}
    {{ template "constructors" $ }}

{{range $field := .Fields}}{{getters $field}}
{{end}}    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        {{ .Name }} that = ({{ .Name }}) o;
{{range $field := .Fields}}{{ equalsBody $field }}
{{end}}
        return true;
    }

    @Override
    public int hashCode() {
        int result = 0;
{{hashcodeBody .Class}}
        return result;
    }

    @Override
    public String toString() {
        return "<{{ .Name }}> {"
{{range $field := .Fields}}{{ toStringBody $field }}{{end}}                + '}';
    }

}`
