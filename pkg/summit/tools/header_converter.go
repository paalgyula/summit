package tools

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type EnumField struct {
	Name  string
	Value string
}

type Enum struct {
	Name   string
	Fields []EnumField
}

func ParseHeaderFile(r io.Reader) []*Enum {
	// Read the input line by line
	scanner := bufio.NewScanner(r)

	var enums []*Enum

	var currentEnum *Enum

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is the start of an enum
		if strings.HasPrefix(line, "enum ") {
			enumName := strings.TrimSpace(strings.TrimPrefix(line, "enum "))
			currentEnum = &Enum{Name: enumName}
			enums = append(enums, currentEnum)

			continue
		}

		// Check if this line is a field in the current enum
		if currentEnum != nil && strings.Contains(line, "=") {
			fieldName := strings.TrimSpace(strings.Split(line, "=")[0])
			fieldName = strings.TrimSpace(strings.Split(fieldName, " ")[0])
			fieldValue := strings.TrimSpace(strings.Split(line, "=")[1])

			// Remove commas from the values, but not the comments
			fieldValue = strings.Replace(fieldValue, ",", "", -1)

			// If contenated (used with base field) then convert the base field to camel too
			if strings.Contains(fieldValue, "+") {
				valueAndBase := strings.Split(fieldValue, " + ")
				fieldValue = fmt.Sprintf("%s + %s", convertToCamelCase(valueAndBase[0]), valueAndBase[1])
			}

			field := EnumField{Name: convertToCamelCase(fieldName), Value: fieldValue}

			currentEnum.Fields = append(currentEnum.Fields, field)

			continue
		}
	}

	return enums
}

type writerConfig struct {
	singleEnum bool
	endField   bool
	enumName   string
}

// WriterOption is a function that can be passed to NewWriter.
type WriterOption func(*writerConfig)

// WithSingleEnum sets whether this is a single enum.
func WithSingleEnum(enumName string) WriterOption {
	return func(c *writerConfig) {
		c.singleEnum = true
		c.enumName = enumName
	}
}

// WithEndField sets when the enum uses reference fields to the previous enum.
func WithEndField(use bool) WriterOption {
	return func(c *writerConfig) {
		c.endField = use
	}
}

// WriteGoSource writes a Go source file.
//
//nolint:funlen
func WriteGoSource(packageName string, enums []*Enum, out io.Writer, opts ...WriterOption) error {
	cfg := writerConfig{
		singleEnum: false,
		endField:   false,
		enumName:   "",
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	bb := &bytes.Buffer{}
	w := bufio.NewWriter(bb)

	w.WriteString("// This file is generated! DO NOT EDIT!\n\n")
	w.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// If this is a single enum, print the header
	if cfg.singleEnum {
		w.WriteString(fmt.Sprintf("type %s int\n\n", cfg.enumName))
	}

	// Generate the output
	for _, enum := range enums {
		if !cfg.singleEnum {
			w.WriteString(fmt.Sprintf("type %s int\n\n", enum.Name))
		}

		w.WriteString("const (\n")

		for idx, field := range enum.Fields {
			if cfg.endField && idx == len(enum.Fields)-1 {
				continue
			}

			enumName := enum.Name
			if cfg.singleEnum {
				enumName = cfg.enumName
			}

			w.WriteString(fmt.Sprintf("\t%s %s = %s\n",
				field.Name,
				enumName, field.Value,
			))
		}

		w.WriteString(")\n\n")

		// Write constant end field
		if cfg.endField {
			endField := enum.Fields[len(enum.Fields)-1]
			w.WriteString(fmt.Sprintf("const %s = %s\n\n", endField.Name, endField.Value))
		}
	}

	// Must flush!
	w.Flush()

	source, err := format.Source(bb.Bytes())
	if err != nil {
		return err
	}

	_, err = out.Write(source)

	return err
}

func toCamelCase(s string) string {
	re := regexp.MustCompile(`([^a-zA-Z0-9]*)([a-zA-Z0-9]+)`)
	parts := re.FindAllStringSubmatch(s, -1)

	var result string

	for _, p := range parts {
		result += cases.Title(language.English).String(strings.ToLower(p[2]))
	}

	return result
}

// convertToCamelCase converts a string in snake_case to camelCase.
//
// input: string in snake_case
// returns: string in camelCase
func convertToCamelCase(input string) string {
	words := strings.Split(strings.ToLower(input), "_")
	for i, word := range words {
		if i == len(words)-1 && isNumeric(word) {
			words[i] = "_" + word

			continue
		}

		words[i] = strings.Title(word)
	}

	return strings.Join(words, "")
}

// isNumeric checks if a given string is a valid decimal number.
//
// str: string to check.
// returns: true if str is a valid decimal number, false otherwise
func isNumeric(str string) bool {
	_, err := strconv.ParseUint(str, 10, 8)

	return err == nil
}
