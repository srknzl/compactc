package main

import (
	"compactc/schema"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"compactc"
)

// flags
var (
	silent bool
	outDir string
)

func init() {
	flag.BoolVar(&silent, "silent", false, "")
	flag.StringVar(&outDir, "output-dir", "./generated", "")
	flag.Usage = func() {
		exp := `Hazelcast Code Generator for Compact Serializer.

positional arguments:
  LANGUAGE              Language to generate codecs for. Possible values are [java cpp cs py ts go]
  SCHEMA_FILE_PATH      Root directory for schema files`
		// nothing to do on err, hence skip
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [-h] [--silent] [--output-dir OUTPUT_DIRECTORY] LANGUAGE SCHEMA_FILE_PATH\n%s", os.Args[0], exp)
		flag.PrintDefaults()
	}
}

var (
	// selected language to generate source for
	lang string
	// path to code generation schema
	schemaPath string
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		exitWithErr("Error: LANGUAGE and SCHEMA_FILE_PATH must be provided\n")
		flag.Usage()
	}
	lang = flag.Arg(0)
	schemaPath = flag.Arg(1)
	if !compactc.IsLangSupported(lang) {
		exitWithErr("Error: Unsupported language, you can provide one of %s\n", strings.Join(compactc.SupportedLangs, ","))
	}
	exitIfPathMissingOrInaccesible(outDir)
	exitIfPathMissingOrInaccesible(schemaPath)
	// validate schemaErr
	yamlSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		exitWithErr("Error: Can not read schema %s\n", err.Error())
	}
	sch, err := schema.ParseSchemaText(string(yamlSchema))
	if err != nil {
		exitWithErr("Error: Can not parse schema %s\n", err.Error())
	}
	classes, err := compactc.GenerateCompactClasses(lang, sch)
	if err != nil {
		exitWithErr("Error: Can not generate compact classes %s\n", err.Error())
	}
	if err = os.MkdirAll(outDir, fs.ModePerm); err != nil {
		exitWithErr("Error: Can not write generated source, path %s, err: %s\n", outDir, err.Error())
	}
	var accumulateErr strings.Builder
	for k, v := range classes {
		if err = os.WriteFile(path.Join(outDir, k.FileName), []byte(v), fs.ModePerm); err != nil {
			accumulateErr.WriteString(err.Error() + "\n")
		}
	}
	if err != nil {
		exitWithErr("Error: Things went wrong while writing genereated source:\n%s\n", err.Error())
	}
	return
}

func exitWithErr(format string, a ...any) {
	// nothing to do if err, so skip
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
	flag.Usage()
	os.Exit(1)
}

func exitIfPathMissingOrInaccesible(path string) {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		exitWithErr("Error: Output directory %s does not exist\n", path)
	} else if err != nil {
		exitWithErr("Error: Can not access output directory %s, err: %s\n", path, err.Error())
	}
}
