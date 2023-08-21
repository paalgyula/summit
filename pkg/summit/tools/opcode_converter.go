package tools

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"regexp"
	"strings"
)

//nolint:lll
const OpcodeHeaderURL = `https://raw.githubusercontent.com/azerothcore/azerothcore-wotlk/master/src/server/game/Server/Protocol/Opcodes.h`

// OpcodeTemplate parsed opcode holder struct.
type OpcodeTemplate struct {
	Name    string
	Value   string
	Comment string
}

func WriteOpcodeSource(packageName string, opcodes []*OpcodeTemplate, out io.Writer) error {
	buf := &bytes.Buffer{}

	w := bufio.NewWriter(buf)
	w.WriteString("// This file is generated! DO NOT EDIT!\n")
	w.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	w.WriteString("// OpCode enumeration of client/server packet opcodes.\n")
	w.WriteString("type OpCode int\n\n")
	w.WriteString("const (\n")

	for _, opcode := range opcodes {
		w.WriteString(fmt.Sprintf("\t%s \tOpCode = %s %s\n", opcode.Name, opcode.Value, opcode.Comment))
	}

	w.WriteString(")\n\n")
	w.Flush() // Always flush..

	bb, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format source: %w", err)
	}

	_, err = out.Write(bb)

	return err
}

func ParseOpcodes(r io.Reader) ([]*OpcodeTemplate, error) {
	var opcodes []*OpcodeTemplate

	scanner := bufio.NewScanner(r)
	pattern := regexp.MustCompile(`((NUM|MSG|CMSG|SMSG)_\w+)\s*=\s*(0x[0-9A-Fa-f]+)(.?\s+(\/\/.+)?)?`)

	for scanner.Scan() {
		line := scanner.Text()
		mm := pattern.FindAllStringSubmatch(line, -1)

		for _, v := range mm {
			opcodes = append(opcodes, &OpcodeTemplate{
				convertOpcodeName(v[1]), v[3], v[5],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return opcodes, nil
}

func convertOpcodeName(code string) string {
	code = strings.NewReplacer(
		"SMSG", "SERVER",
		"CMSG", "CLIENT",
	).Replace(code)

	return toCamelCase(code)
}

func Fetch(url string) (io.Reader, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the body into memory
	buf := new(bytes.Buffer)

	if _, err = io.Copy(buf, resp.Body); err != nil {
		return nil, err
	}

	return buf, nil
}
