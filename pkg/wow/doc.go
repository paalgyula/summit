// This package contains the client/server packet opcodes,
// tools for creating the data layer of the Summit emulator.
// Some of the code here is auto-generated, the source is available at
// the pkg/summit/tools package (https://github.com/paalgyula/summit/tree/master/pkg/summit/tools)
//
// To run the toolset and the code generators, you should first install the required packages by
// running the following command in the root of the project: `make install` this command will
// build the 'datagen' binary and copies to your GOBIN directory. After you can run the go generate command
// or simply invoke it with `make gen` command which will do the rest for you. The
// pre-generated opcodes already commited to the repo so you don't need to re-generate them, but
// if you want to play with you can do it manually.
//
//go:generate datagen opcodes -p wow -o opcodes.gen.go
//go:generate stringer -type=OpCode -output=opcodes_string.go
package wow
