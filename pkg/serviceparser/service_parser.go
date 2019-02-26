package serviceparser

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()
var sugarLogger = logger.Sugar()

type importContainer struct {
	LocalName    string `json:"local_name"`
	ImportPath   string `json:"import_path"`
	DependentPkg string `json:"dependent_pkg"`
}

// AllPkgFunc variable contains all the services mapped to their corresponding functions.
var AllPkgFunc = make(map[string]map[string][]string)

// AllPkgImports contains all the external dependencies.
var AllPkgImports = make(map[string]interface{})

// ParseService parses a service and dumps all its functions to a JSON
func ParseService(serviceName string, root string, destdir string) {
	sugarLogger.Info("Walking: ", root)
	AllPkgFunc[serviceName] = make(map[string][]string)
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		// Do not visit git dir.
		if f.IsDir() && (f.Name() == ".git" || f.Name() == "vendor") {
			return filepath.SkipDir
		}
		// Our logic is not for files.
		if !f.IsDir() {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseDir(fset,
			path,
			nil, parser.ParseComments)
		if err != nil {
			sugarLogger.Fatal(err)
		}

		for pkg, ast := range node {
			pkgFunctions, pkgImports := parseServiceAST(ast, fset, pkg)
			AllPkgFunc[serviceName][pkg] = pkgFunctions
			AllPkgImports[serviceName] = pkgImports
		}
		return nil
	})
	if err != nil {
		sugarLogger.Fatal(err)
	}
	packageJSON, err := json.Marshal(AllPkgFunc[serviceName])
	if err != nil {
		sugarLogger.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(destdir, serviceName+".json"), packageJSON, 0644)
	if err != nil {
		panic(err)
	}
}

func parseImportNode(imp *ast.ImportSpec, pkg string) importContainer {
	var impName string
	if imp.Name != nil {
		impName = imp.Name.Name
	} else {
		_, impName = filepath.Split(imp.Path.Value)
	}
	ic := importContainer{
		LocalName:    impName,
		ImportPath:   imp.Path.Value,
		DependentPkg: pkg,
	}
	sugarLogger.Infof("%v\n", ic)
	return ic
}

func parseServiceAST(node ast.Node, fset *token.FileSet, pkg string) ([]string, []importContainer) {
	var functions []string
	var imports []importContainer
	ast.Inspect(node, func(n ast.Node) bool {

		// Find Functions
		switch fnOrImp := n.(type) {
		case *ast.FuncDecl:
			functions = append(functions, fnOrImp.Name.Name)
		case *ast.ImportSpec:
			imports = append(imports, parseImportNode(fnOrImp, pkg))
		}
		return true
	})

	return functions, imports
}