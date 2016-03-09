package apcompiler

import (
	"go/ast"
	"github.com/alangpierce/apgo/apast"
	"go/token"
	"github.com/alangpierce/apgo/apruntime"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type CompileCtx struct {
	NativePackages map[string]*apruntime.NativePackage
}

func CompilePackage(ctx CompileCtx, pack *ast.Package) *apast.Package {
	funcs := make(map[string]*apast.FuncDecl)
	for _, file := range pack.Files {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				funcs[decl.Name.Name] = CompileFuncDecl(ctx, decl)
			}
		}
	}
	return &apast.Package{
		funcs,
	}
}

func CompileFuncDecl(ctx CompileCtx, funcDecl *ast.FuncDecl) *apast.FuncDecl {
	paramNames := []string{}
	for _, param := range funcDecl.Type.Params.List {
		if param.Names == nil {
			paramNames = append(paramNames, "_")
		} else if len(param.Names) == 1 {
			paramNames = append(paramNames, param.Names[0].Name)
		} else {
			panic("Unexpected number of parameter names.")
		}
	}
	return &apast.FuncDecl{
		CompileStmt(ctx, funcDecl.Body),
		paramNames,
	}
}

func CompileStmt(ctx CompileCtx, stmt ast.Stmt) apast.Stmt {
	switch stmt := stmt.(type) {
	//case *ast.BadStmt:
	//	return nil
	//case *ast.DeclStmt:
	//	return nil
	//case *ast.EmptyStmt:
	//	return nil
	//case *ast.LabeledStmt:
	//	return nil
	case *ast.ExprStmt:
		return &apast.ExprStmt{
			compileExpr(ctx, stmt.X),
		}
	//case *ast.SendStmt:
	//	return nil
	//case *ast.IncDecStmt:
	//	return nil
	case *ast.AssignStmt:
		if stmt.Tok == token.DEFINE || stmt.Tok == token.ASSIGN {
			lhs := []apast.Expr{}
			rhs := []apast.Expr{}
			for _, lhsExpr := range stmt.Lhs {
				lhs = append(lhs, compileExpr(ctx, lhsExpr))
			}
			for _, rhsExpr := range stmt.Rhs {
				rhs = append(rhs, compileExpr(ctx, rhsExpr))
			}
			return &apast.AssignStmt{
				lhs,
				rhs,
			}
		} else {
			if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
				panic("Unexpected multiple assign")
			}
			// TODO: We should only evaluate the left side once,
			// e.g. array index values.
			compiledLhs := compileExpr(ctx, stmt.Lhs[0])
			return &apast.AssignStmt{
				[]apast.Expr{compiledLhs},
				[]apast.Expr{
					&apast.FuncCallExpr{
						&apast.LiteralExpr{
							apruntime.AssignBinaryOperators[stmt.Tok],
						},
						[]apast.Expr{
							compiledLhs,
							compileExpr(ctx, stmt.Rhs[0]),
						},
					},
				},
			}
		}
	//case *ast.GoStmt:
	//	return nil
	//case *ast.DeferStmt:
	//	return nil
	case *ast.ReturnStmt:
		resultsExprs := []apast.Expr{}
		for _, result := range stmt.Results {
			resultsExprs = append(resultsExprs, compileExpr(ctx, result))
		}
		return &apast.ReturnStmt{
			resultsExprs,
		}
	//case *ast.BranchStmt:
	//	return nil
	case *ast.BlockStmt:
		stmts := []apast.Stmt{}
		for _, subStmt := range stmt.List {
			stmts = append(stmts, CompileStmt(ctx, subStmt))
		}
		return &apast.BlockStmt{
			stmts,
		}
	case *ast.IfStmt:
		var result apast.IfStmt
		if stmt.Init != nil {
			result.Init = CompileStmt(ctx, stmt.Init)
		} else {
			result.Init = &apast.EmptyStmt{}
		}
		result.Cond = compileExpr(ctx, stmt.Cond)
		result.Body = CompileStmt(ctx, stmt.Body)
		if stmt.Else != nil {
			result.Else = CompileStmt(ctx, stmt.Else)
		} else {
			result.Else = &apast.EmptyStmt{}
		}
		return &result
	//case *ast.CaseClause:
	//	return nil
	//case *ast.SwitchStmt:
	//	return nil
	//case *ast.TypeSwitchStmt:
	//	return nil
	//case *ast.CommClause:
	//	return nil
	//case *ast.SelectStmt:
	//	return nil
	//case *ast.ForStmt:
	//	return nil
	//case *ast.RangeStmt:
	//	return nil
	default:
		panic(fmt.Sprint("Statement compile not implemented: ", reflect.TypeOf(stmt)))
	}
}

func compileExpr(ctx CompileCtx, expr ast.Expr) apast.Expr {
	switch expr := expr.(type) {
	//case *ast.BadExpr:
	//	return nil
	case *ast.Ident:
		return &apast.IdentExpr{
			expr.Name,
		}
	//case *ast.Ellipsis:
	//	return nil
	case *ast.BasicLit:
		return &apast.LiteralExpr{
			reflect.ValueOf(parseLiteral(expr.Value, expr.Kind)),
		}
	//case *ast.FuncLit:
	//	return nil
	//case *ast.CompositeLit:
	//	return nil
	//case *ast.ParenExpr:
	//	return nil
	case *ast.SelectorExpr:
		if leftSide, ok := expr.X.(*ast.Ident); ok {
			nativePackage := ctx.NativePackages[leftSide.Name]
			if nativePackage == nil {
				panic(fmt.Sprint("Unknown package ", leftSide.Name))
			}
			funcVal := nativePackage.Funcs[expr.Sel.Name]
			if nativePackage == nil {
				panic(fmt.Sprint("Unknown function ", expr.Sel.Name))
			}
			return &apast.LiteralExpr{reflect.ValueOf(funcVal)}
		}
		panic(fmt.Sprint("Selector not found ", expr))
		return nil
	//case *ast.IndexExpr:
	//	return nil
	//case *ast.SliceExpr:
	//	return nil
	//case *ast.TypeAssertExpr:
	//	return nil
	case *ast.CallExpr:
		compiledArgs := []apast.Expr{}
		for _, arg := range expr.Args {
			compiledArgs = append(compiledArgs, compileExpr(ctx, arg))
		}
		return &apast.FuncCallExpr{
			compileExpr(ctx, expr.Fun),
			compiledArgs,
		}
	//case *ast.StarExpr:
	//	return nil
	//case *ast.UnaryExpr:
	//	return nil
	case *ast.BinaryExpr:
		if op, ok := apruntime.BinaryOperators[expr.Op]; ok {
			return &apast.FuncCallExpr{
				&apast.LiteralExpr{
					op,
				},
				[]apast.Expr{compileExpr(ctx, expr.X), compileExpr(ctx, expr.Y)},
			}
		} else {
			panic(fmt.Sprint("Operator not implemented: ", expr.Op))
		}
	//case *ast.KeyValueExpr:
	//	return nil
	//
	//case *ast.ArrayType:
	//	return nil
	//case *ast.StructType:
	//	return nil
	//case *ast.FuncType:
	//	return nil
	//case *ast.InterfaceType:
	//	return nil
	//case *ast.MapType:
	//	return nil
	//case *ast.ChanType:
	//	return nil
	default:
		panic(fmt.Sprint("Expression compile not implemented: ", reflect.TypeOf(expr)))
		return nil
	}
}

// parseLiteral takes a primitive literal and returns it as a value.
func parseLiteral(val string, kind token.Token) interface{} {
	switch kind {
	case token.IDENT:
		panic("TODO")
		return nil
	case token.INT:
		// Note that base 0 means that octal and hex literals are also
		// handled.
		result, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			panic(err)
		}
		return result
	case token.FLOAT:
		panic("TODO")
		return nil
	case token.IMAG:
		panic("TODO")
		return nil
	case token.CHAR:
		panic("TODO")
		return nil
	case token.STRING:
		return parseString(val)
	default:
		fmt.Print("Unrecognized kind: ", kind)
		return nil
	}
}

func parseString(codeString string) string {
	strWithoutQuotes := codeString[1:len(codeString) - 1]
	// TODO: Replace with an implementation that properly escapes
	// everything.
	return strings.Replace(strWithoutQuotes, "\\n", "\n", -1)
}