package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// errorResponders are the helpers that end a request by writing an error body.
//
// Calling one of these commits the response: a status and a JSON error document are
// already on the wire, so any statement that follows writes a *second* body into the
// same response.
var errorResponders = map[string]bool{
	"ServerErrorResponse":                true,
	"BadRequestResponse":                 true,
	"NotFoundResponse":                   true,
	"MethodNotAllowedResponse":           true,
	"FailedValidationResponse":           true,
	"EditConflictResponse":               true,
	"NotPermittedResponse":               true,
	"InvalidCredentialsResponse":         true,
	"InvalidAuthenticationTokenResponse": true,
	"AuthenticationRequiredResponse":     true,
	"InactiveAccountResponse":            true,
	"RateLimitExceededResponse":          true,
}

// An error response must be the last thing a block does.
//
// This is a lint, not a unit test, because the bug it catches is invisible in review and
// silent at runtime. `GET /api/patrulje` once answered with an error envelope *followed
// by* `{"teams": null}` — two JSON documents in one body, which no client can parse — for
// no reason other than a missing `return`. Six more handlers had the same omission, and
// three of those were Excel exports that went on to build and send an entire spreadsheet
// from a nil slice, so an operator downloaded a plausible, silently empty file.
//
// Walking the AST rather than grepping matters: the naive grep over this package returns
// well over a hundred hits, nearly all of them the trailing `WriteJSON` error handler
// where falling through is harmless, which is precisely why the real six went unnoticed
// for so long. Here the question asked is the exact one: within this block, is there a
// statement after the error response?
func TestErrorResponsesAreFollowedByReturn(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			var list []ast.Stmt
			switch block := n.(type) {
			case *ast.BlockStmt:
				list = block.List
			case *ast.CaseClause:
				// A case clause needs no `return`: Go does not fall through. Its body is
				// still checked, so an error response mid-case is caught.
				list = block.Body
			default:
				return true
			}

			for i, stmt := range list {
				if i == len(list)-1 {
					continue // nothing follows in this block, so nothing writes a second body
				}
				following := len(list) - i - 1

				// The shape all seven bugs actually had: a guard that responds and then
				// simply ends, letting the handler carry on to write its success body.
				//
				//	if err != nil {
				//	    app.ServerErrorResponse(w, r, err)
				//	}
				//	err = app.WriteJSON(...)   <- second body
				//
				// The identical guard as the *last* statement of a handler is the correct
				// and very common trailing WriteJSON check, which is why this asks about
				// the enclosing block rather than about the guard alone.
				if ifStmt, ok := stmt.(*ast.IfStmt); ok && ifStmt.Else == nil && !terminates(list[i+1:i+2]) {
					body := ifStmt.Body.List
					if len(body) > 0 {
						if name, ok := errorResponseCall(body[len(body)-1]); ok {
							pos := fset.Position(body[len(body)-1].Pos())
							t.Errorf("%s:%d: %s ends this guard, but %d statement(s) follow it; add a return, or the response gets a second body",
								filepath.Base(pos.Filename), pos.Line, name, following)
						}
					}
				}

				name, ok := errorResponseCall(stmt)
				if !ok {
					continue
				}
				if terminates(list[i+1 : i+2]) {
					continue // the very next statement ends the handler, which is the fix
				}
				if _, isCase := n.(*ast.CaseClause); isCase && terminates(list[i+1:]) {
					continue
				}
				pos := fset.Position(stmt.Pos())
				t.Errorf("%s:%d: %s is followed by %d more statement(s) in this block; add a return, or the response gets a second body",
					filepath.Base(pos.Filename), pos.Line, name, following)
			}
			return true
		})
	}
}

// errorResponseCall reports whether stmt is a bare call to one of the error responders.
func errorResponseCall(stmt ast.Stmt) (string, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return "", false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if !errorResponders[sel.Sel.Name] {
		return "", false
	}
	return sel.Sel.Name, true
}

// terminates reports whether the statements end control flow, so a following statement
// cannot in fact run.
func terminates(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return true
	}
	switch stmts[len(stmts)-1].(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	}
	return false
}
