package main

import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

//go:embed opcodes.go.tpl
var handlerTemplate string

var MoveOpcodes = []string{
	"OpCodeClientMoveHeartbeat",
	"OpCodeClientMoveSetFacing",
	"OpCodeClientMoveStartBackward",
	"OpCodeClientMoveStartForward",
	"OpCodeClientMoveStartStrafeLeft",
	"OpCodeClientMoveStartStrafeRight",
	"OpCodeClientMoveStartTurnLeft",
	"OpCodeClientMoveStartTurnRight",
	"OpCodeClientMoveStop",
	"OpCodeClientMoveStopStrafe",
	"OpCodeClientMoveStopTurn",
}

type packetHandler struct {
	Name   string
	Packet string
}

func findFunctions() []packetHandler {
	fset := token.NewFileSet()
	pf, err := parser.ParseDir(fset, "server/world/packet", nil, parser.ParseComments|parser.DeclarationErrors)
	if err != nil {
		panic(err)
	}

	var funcs []packetHandler

	fn := ""
	ast.Inspect(pf["packet"], func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			fn = t.Name.String()
		case *ast.ReturnStmt:
			if sel, ok := (t.Results[0]).(*ast.SelectorExpr); ok {
				if strings.HasPrefix(fn, "Client") {

					if strings.Contains(sel.Sel.Name, "MoveOpCode") {
						break
					}

					funcs = append(funcs, packetHandler{
						Name: fn,
						// Packet: fmt.Sprintf("%s", sel.Sel.Name),
						Packet: sel.Sel.Name,
					})
				}
			}
		default:
			// fmt.Printf("%T\n", n)
		}

		return true
	})

	for _, oc := range MoveOpcodes {
		funcs = append(funcs, packetHandler{
			Name:   "ClientMove",
			Packet: oc,
		})
	}

	return funcs
}

func main() {
	funcs := findFunctions()

	// filePath := "server/world/data/static/opcodes.go"
	// of, err := os.Open(filePath)
	// if err != nil {
	// 	panic("opcodes file not found")
	// }

	// src, _ := io.ReadAll(of)

	// fs := token.NewFileSet()
	// pf, err := parser.ParseFile(fs, "", src, parser.ParseComments)
	// if err != nil {
	// 	panic(err)
	// }

	// ast.Inspect(pf, func(n ast.Node) bool {
	// 	switch t := n.(type) {
	// 	case *ast.ValueSpec:
	// 		fmt.Printf("%s %+v", t.Doc.Text(), t.Values[0])
	// 	default:
	// 		fmt.Printf("Not handled type: %T\n", n)
	// 	}
	// 	return true
	// })
	// generateHandlers()

	tpl, err := template.New("t").Parse(handlerTemplate)
	if err != nil {
		panic(err)
	}

	f, _ := os.Create("server/world/opcodes.gen.go")
	tpl.Execute(f, map[string]interface{}{
		"Opcodes": funcs,
	})
}

// func generateHandlers() {
// 	tpl, err := template.New("t").Parse(handlerTemplate)
// 	if err != nil {
// 		panic(err)
// 	}

// 	f, _ := os.Create("server/world/opcodes.gen.go")
// 	tpl.Execute(f, map[string]interface{}{
// 		"Opcodes": funcs,
// 	})
// }
