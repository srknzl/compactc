package compactc

import (
	"fmt"
	"strings"

	"compactc/common"
	"compactc/java"
)

type Lang string

const (
	JAVA       = "java"
	PYTHON     = "py"
	TYPESCRIPT = "ts"
	CPP        = "cpp"
	GO         = "go"
	CSHARP     = "cs"
)

var SupportedLangs = []string{JAVA, PYTHON, TYPESCRIPT, CPP, GO, CSHARP}

func IsLangSupported(lang string) bool {
	lang = strings.ToLower(lang)
	for _, sl := range SupportedLangs {
		if lang == sl {
			return true
		}
	}
	return false
}

func GenerateCompactClasses(lang string, schema common.Schema) (map[common.ClassAndFileName]string, error) {
	// compactSchema name to generated compactSchema
	classes := make(map[common.ClassAndFileName]string)
	switch lang {
	case JAVA:
		javaClasses := java.Generate(schema)
		for jc := range javaClasses {
			classes[common.ClassAndFileName{
				FileName:  fmt.Sprintf("%s.java", jc),
				ClassName: jc,
			}] = javaClasses[jc]
		}
	case PYTHON:
		panic(any("implement me"))
	case TYPESCRIPT:
		panic(any("implement me"))
	case CPP:
		panic(any("implement me"))
	case GO:
		panic(any("implement me"))
	case CSHARP:
		panic(any("implement me"))
	default:
		return nil, fmt.Errorf("unsupported langugage")
	}
	return classes, nil
}
