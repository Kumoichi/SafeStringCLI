package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/tools/go/packages"
)

type SafeStringFixer struct {
	fset     *token.FileSet
	info     *types.Info
	modified bool
}

func NewSafeStringFixer() *SafeStringFixer {
	return &SafeStringFixer{
		fset: token.NewFileSet(),
		info: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Uses:  make(map[*ast.Ident]types.Object),
		},
	}
}

func (sf *SafeStringFixer) processFile(filename string) error {
	fmt.Printf("Processing file: %s\n", filename)
	
	// Parse the file
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	file, err := parser.ParseFile(sf.fset, filename, src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %v", filename, err)
	}

	// Simple type checking without full package resolution for now
	// We'll identify string types by examining the AST structure
	sf.modified = false
	
	// Add errorsx import if we make modifications
	var hasErrorsxImport bool
	for _, imp := range file.Imports {
		if strings.Contains(imp.Path.Value, "errorsx") {
			hasErrorsxImport = true
			break
		}
	}

	// Transform the AST
	ast.Inspect(file, sf.visitNode)

	if sf.modified {
		// Add errorsx import if not present and we made modifications
		if !hasErrorsxImport {
			sf.addErrorsxImport(file)
		}

		// Generate output filename
		outputPath := sf.generateOutputPath(filename)
		
		// Write the modified file
		if err := sf.writeFile(outputPath, file); err != nil {
			return fmt.Errorf("failed to write output file %s: %v", outputPath, err)
		}
		
		fmt.Printf("Generated: %s\n", outputPath)
	} else {
		fmt.Printf("No modifications needed for %s\n", filename)
	}

	return nil
}

func (sf *SafeStringFixer) visitNode(n ast.Node) bool {
	// Look for function calls
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return true
	}

	// Check if it's fmt.Errorf call
	if !sf.isFmtErrorf(call) {
		return true
	}

	// Must have at least 2 arguments (format string + at least one arg)
	if len(call.Args) < 2 {
		return true
	}

	// Check if first argument is a format string with %v or %s
	formatStr := sf.extractFormatString(call.Args[0])
	if formatStr == "" {
		return true
	}

	// Check if format contains %v or %s
	if !strings.Contains(formatStr, "%v") && !strings.Contains(formatStr, "%s") {
		return true
	}

	// Process arguments starting from index 1 (skip format string)
	for i := 1; i < len(call.Args); i++ {
		if sf.shouldWrapArgument(call.Args[i]) {
			// Wrap the argument with errorsx.SafeString
			call.Args[i] = sf.wrapWithSafeString(call.Args[i])
			sf.modified = true
		}
	}

	return true
}

func (sf *SafeStringFixer) isFmtErrorf(call *ast.CallExpr) bool {
	// Check if it's a selector expression (pkg.func)
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Check if selector is "Errorf"
	if sel.Sel.Name != "Errorf" {
		return false
	}

	// Check if the package is "fmt"
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "fmt"
}

func (sf *SafeStringFixer) extractFormatString(expr ast.Expr) string {
	// Handle basic string literals
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		// Remove quotes and return the string content
		s := lit.Value
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return ""
}

func (sf *SafeStringFixer) shouldWrapArgument(expr ast.Expr) bool {
	// Check if it's already wrapped with SafeString
	if call, ok := expr.(*ast.CallExpr); ok {
		if sf.isSafeStringCall(call) {
			return false // Already wrapped
		}
	}

	// For simplicity, we'll assume that identifiers (variables) are strings
	// In a full implementation, we'd use type checking
	if _, ok := expr.(*ast.Ident); ok {
		return true
	}

	// Also handle selector expressions (e.g., obj.field)
	if _, ok := expr.(*ast.SelectorExpr); ok {
		return true
	}

	return false
}

func (sf *SafeStringFixer) isSafeStringCall(call *ast.CallExpr) bool {
	// Check if it's errorsx.SafeString call
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if sel.Sel.Name != "SafeString" {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "errorsx"
}

func (sf *SafeStringFixer) wrapWithSafeString(expr ast.Expr) ast.Expr {
	// Create errorsx.SafeString(expr)
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "errorsx"},
			Sel: &ast.Ident{Name: "SafeString"},
		},
		Args: []ast.Expr{expr},
	}
}

func (sf *SafeStringFixer) addErrorsxImport(file *ast.File) {
	// Create new import spec
	errorxsImport := &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"github.com/Kumoichi/SafeStringCLI/errorsx"`,
		},
	}

	// Add to imports
	if file.Imports == nil {
		// Create new import declaration
		importDecl := &ast.GenDecl{
			Tok: token.IMPORT,
			Specs: []ast.Spec{errorxsImport},
		}
		file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
	} else {
		// Find existing import declaration and add to it
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
				genDecl.Specs = append(genDecl.Specs, errorxsImport)
				break
			}
		}
	}
}

func (sf *SafeStringFixer) generateOutputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+"_fix"+ext)
}

func (sf *SafeStringFixer) writeFile(filename string, file *ast.File) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return format.Node(f, sf.fset, file)
}

func (sf *SafeStringFixer) processDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip already processed files (ending with _fix.go)
		if strings.HasSuffix(path, "_fix.go") {
			return nil
		}

		// Skip Go test files (ending with _test.go)
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return sf.processFile(path)
	})
}

func main() {
	fmt.Println("SafeString CLI Fixer")
	fmt.Println("===================")

	fixer := NewSafeStringFixer()
	
	// Process current directory
	if err := fixer.processDirectory("."); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Processing complete!")
}