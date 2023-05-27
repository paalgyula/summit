package tools

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpcodeFetcher(t *testing.T) {
	t.Skip("don't flood the server...")

	fileName := "Opcodes.h"
	f, _ := os.Create(fileName)
	content, err := Fetch(OpcodeHeaderURL)
	assert.NoError(t, err)

	_, err = io.Copy(f, content)
	assert.NoError(t, err)
}

func TestOpcodeParser(t *testing.T) {
	fileName := "Opcodes.h"
	f, _ := os.Open(fileName)
	opcodes, err := ParseOpcodes(f)

	assert.NoError(t, err)
	assert.NotNil(t, opcodes)

	err = WriteOpcodeSource("wow", opcodes, os.Stdout)
	assert.NoError(t, err)
}
