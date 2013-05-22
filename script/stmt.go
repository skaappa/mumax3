package script

import (
	"go/ast"
)

// a statement can be executed
type Stmt interface {
	Exec()
}

// compiles a statement
func (w *World) compileStmt(st ast.Stmt) Stmt {
	switch concrete := st.(type) {
	default:
		panic(err("not allowed:", st))
	case *ast.AssignStmt:
		return w.compileAssignStmt(concrete)
	}
}
