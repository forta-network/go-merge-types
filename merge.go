package merge

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/forta-network/go-merge-types/utils"
	"gopkg.in/yaml.v3"
)

func Run(configPath string) (*MergeConfig, []byte, error) {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, err
	}

	var config MergeConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		return &config, nil, err
	}

	// fix package source dirs relative to the config path
	for _, source := range config.Sources {
		source.Package.SourceDir = utils.RelativePath(configPath, source.Package.SourceDir)
	}

	b, err = Generate(&config)
	if err != nil {
		return &config, nil, err
	}

	return &config, b, nil
}

func Generate(config *MergeConfig) ([]byte, error) {
	var pkgs []*ast.Package
	for _, source := range config.Sources {
		pkgs = append(pkgs, LoadPackage(source.Package.SourceDir))
	}

	var impls []*SourceImplementation
	for i, pkg := range pkgs {
		impls = append(impls, FindImplementation(pkg, config.Sources[i].Type))
	}

	return mergeAndGenerate(config, impls)
}

func LoadPackage(pkgDir string) *ast.Package {
	fset := token.NewFileSet()
	foundPkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range foundPkgs {
		return p
	}
	log.Fatalf("no package found at: %s", pkgDir)
	return nil
}

type SourceImplementation struct {
	Package     *ast.Package
	Object      *ast.Object
	Constructor *ast.FuncDecl
	Methods     []*ast.FuncDecl
	Types       []*ast.TypeSpec
	Imports     []*ast.ImportSpec
}

func (sourceImpl *SourceImplementation) GetType(expr ast.Expr) (*ast.TypeSpec, bool) {
	searchType := deref(typeString("", "", expr))
	for _, typ := range sourceImpl.Types {
		if typ.Name.Name == searchType {
			return typ, true
		}
	}
	return nil, false
}

func (sourceImpl *SourceImplementation) GetImport(input string) (string, bool) {
	parts := strings.Split(input, ".")
	if len(parts) == 2 {
		input = strings.Trim(parts[0], "*")
	}

	for _, imp := range sourceImpl.Imports {
		// find by name (alias)
		if imp.Name != nil && imp.Name.Name == input {
			return fmt.Sprintf("%s %s", imp.Name.Name, imp.Path.Value), true
		}

		// find from path
		parts := strings.Split(strings.Trim(imp.Path.Value, `"`), "/")
		altName := parts[len(parts)-1]
		if altName == input {
			return imp.Path.Value, true
		}
	}

	return "", false
}

func FindImplementation(pkg *ast.Package, implName string) *SourceImplementation {
	var impl SourceImplementation
	impl.Package = pkg

	constructorName := fmt.Sprintf("New%s", implName)

	for _, file := range pkg.Files {
		// collect all imports from all files
		impl.Imports = append(impl.Imports, file.Imports...)

		if file.Scope != nil {
			var ok bool
			obj, ok := file.Scope.Objects[implName]
			if ok && impl.Object == nil {
				impl.Object = obj
			}
		}
	}
	if impl.Object == nil {
		log.Fatalf("implementation not found in %s", pkg.Name)
		return nil
	}

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				// find the constructor
				if funcDecl.Recv == nil && funcDecl.Name.Name == constructorName {
					impl.Constructor = funcDecl
					continue
				}

				// find the implemented methods
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
					recvList := funcDecl.Recv.List
					if recvList[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name == implName {
						impl.Methods = append(impl.Methods, funcDecl)
					}
				}
			}
		}

		for _, object := range file.Scope.Objects {
			if typeDecl, ok := object.Decl.(*ast.TypeSpec); ok {
				impl.Types = append(impl.Types, typeDecl)
			}
		}
	}

	if impl.Constructor == nil {
		log.Fatalf("constructor %s was not found for type %s in package %s", constructorName, implName, pkg.Name)
		return nil
	}

	return &impl
}

func mergeAndGenerate(config *MergeConfig, sourceImpls []*SourceImplementation) ([]byte, error) {
	// fix empty package aliases: find package name from ast and append source index i to the name.
	for i, source := range config.Sources {
		if len(source.Package.Alias) > 0 {
			continue
		}
		sourceImpl := sourceImpls[i]
		source.Package.Alias = fmt.Sprintf("%s_%d", sourceImpl.Package.Name, i+1)
	}

	// find output type init args
	for i, sourceImpl := range sourceImpls {
		params := sourceImpl.Constructor.Type.Params
		if params == nil {
			continue
		}
		pkgName := config.Sources[i].Package.Alias
		for _, param := range params.List {
			foundParam, ok := isNewParam(pkgName, i, param, config.Output.InitArgs)
			foundParam.SourceIndex = i
			if ok {
				config.Output.InitArgs = append(config.Output.InitArgs, foundParam)
			}

			// add source type init args
			config.Sources[i].InitArgs = append(config.Sources[i].InitArgs, foundParam)
		}
	}

	// collect all variations of all methods and their input & output types under bucket methods

	allMethods := make([]*Method, 0)

	for i, sourceImpl := range sourceImpls {
		pkgName := config.Sources[i].Package.Alias

		for _, sourceMethod := range sourceImpl.Methods {
			// create a method variation
			var variation Variation
			variation.SourceIndex = i
			variation.Tag = config.Sources[i].Tag
			methodName := sourceMethod.Name.Name
			variation.Name = methodName

			// find out variation method return type

			var resultsList []*ast.Field
			if sourceMethod.Type.Results != nil {
				resultsList = sourceMethod.Type.Results.List
			}

			var ret *ast.Field
			switch len(resultsList) {
			case 0:
				variation.NoReturn = true

			case 1:
				v := resultsList[0]
				if types.ExprString(v.Type) == "error" {
					variation.OnlyError = true
				} else {
					ret = v
				}

			case 2:
				v1, v2 := resultsList[0], resultsList[1]
				if types.ExprString(v2.Type) != "error" {
					log.Printf("warning: expected %s.%s.%s to return (<any>, error) - ignoring\n", pkgName, sourceImpl.Object.Name, methodName)
					continue
				}
				ret = v1

			default:
				log.Printf("warning: %s.%s.%s has unsupported return list - ignoring\n", pkgName, sourceImpl.Object.Name, methodName)
				continue
			}

			// find the bucket (merged) method
			// if doesn't exist, create
			var method *Method
			for _, m := range allMethods {
				if m.Name == methodName {
					method = m
					break
				}
			}
			if method == nil {
				method = &Method{
					Name: methodName,
					ReturnType: ReturnType{
						Name: methodName + "Output",
					},
				}
				allMethods = append(allMethods, method)
			}

			// add this definition as a variation of the method
			method.Variations = append(method.Variations, &variation)

			// set args
			for _, param := range sourceMethod.Type.Params.List {
				field := convertField(pkgName, i, param)
				field.SourceIndex = i
				variation.Args = append(variation.Args, field)
			}

			if ret == nil {
				continue
			}

			// anonymous struct
			structType, ok := ret.Type.(*ast.StructType)
			if ok {
				variation.MergeReturnedStruct = true
				for _, param := range structType.Fields.List {
					variation.ReturnedFields = append(variation.ReturnedFields, &Field{
						SourceIndex: i,
						Name:        param.Names[0].Name,
						Type:        typeString("", pkgName, param.Type),
					})
				}
				continue
			}

			// local struct
			if isLocalType(true, ret.Type) {
				localType, ok := sourceImpl.GetType(ret.Type)
				if !ok {
					log.Fatalf("local type not found: %s", typeString(pkgName, "", ret.Type))
				}
				if structType, ok := localType.Type.(*ast.StructType); ok {
					if !hasUnexportedField(structType.Fields.List) {
						// local struct with exported fields
						variation.MergeReturnedStruct = true
						for _, param := range structType.Fields.List {
							variation.ReturnedFields = append(variation.ReturnedFields, &Field{
								SourceIndex: i,
								Name:        param.Names[0].Name,
								Type:        typeString("", pkgName, param.Type),
							})
						}
					} else {
						// local struct with unexported fields
						variation.ReturnedFields = append(variation.ReturnedFields, &Field{
							SourceIndex: i,
							Name:        pkgNameToMethodPrefix(pkgName) + "Result",
							Type:        typeString("", pkgName, ret.Type),
						})
					}
				}
			}

			// local non-struct or imported
			if len(variation.ReturnedFields) == 0 {
				retType := typeString("", pkgName, ret.Type)
				retNameSuffix := deref(retType)
				if strings.Contains(retNameSuffix, ".") {
					retNameSuffix = strings.Title(strings.Join(strings.Split(retNameSuffix, "."), ""))
				}
				variation.ReturnedFields = append(variation.ReturnedFields, &Field{
					SourceIndex: i,
					Name:        "Value",
					Type:        retType,
				})
			}

		}
	}

	// construct all bucket method inputs and outputs
	for _, method := range allMethods {
		for _, variation := range method.Variations {
			// merge args
			method.Args = mergeFields(variation.Args, method.Args)

			// merge return fields
			method.ReturnType.Fields = mergeFields(variation.ReturnedFields, method.ReturnType.Fields)
		}

		// decide on return value
		switch len(method.ReturnType.Fields) {
		case 0:
			method.NoReturn = true
		case 1:
			method.SingleReturn = true
			method.ReturnType.Name = method.ReturnType.Fields[0].Type
		}
	}

	// merge all imports for inputs and outputs
	for _, method := range allMethods {
		for _, field := range method.Args {
			impl := sourceImpls[field.SourceIndex]
			imp, ok := impl.GetImport(field.Type)
			if ok {
				config.Output.Imports = mergeImport(imp, config.Output.Imports)
			}
		}
		for _, field := range method.ReturnType.Fields {
			impl := sourceImpls[field.SourceIndex]
			imp, ok := impl.GetImport(field.Type)
			if ok {
				config.Output.Imports = mergeImport(imp, config.Output.Imports)
			}
		}
	}

	// merge init arg imports
	for _, field := range config.Output.InitArgs {
		impl := sourceImpls[field.SourceIndex]
		imp, ok := impl.GetImport(field.Type)
		if ok {
			config.Output.Imports = mergeImport(imp, config.Output.Imports)
		}
	}

	// set all methods in the config
	config.Output.Methods = allMethods

	// rewrite some names: constructor (init) args, method names, method inputs, method outputs
	rewriter := config.Output.Rewrite
	for _, initArg := range config.Output.InitArgs {
		initArg.Name = rewriter.Rewrite(initArg.Name)
		initArg.Type = rewriter.Rewrite(initArg.Type)
	}
	for _, method := range config.Output.Methods {
		method.Name = rewriter.Rewrite(method.Name)
		for _, arg := range method.Args {
			arg.Name = rewriter.Rewrite(arg.Name)
			arg.Type = rewriter.Rewrite(arg.Type)
		}
		method.ReturnType.Name = rewriter.Rewrite(method.ReturnType.Name)
		for _, field := range method.ReturnType.Fields {
			field.Name = rewriter.Rewrite(field.Name)
			field.Type = rewriter.Rewrite(field.Type)
		}
	}

	// finally, execute the config on the template and return
	buffer := new(bytes.Buffer)
	tmpl := template.Must(template.New("").Parse(codeTemplate))
	if err := tmpl.Execute(buffer, config); err != nil {
		return nil, err
	}
	return []byte(strings.TrimSpace(string(buffer.Bytes()))), nil
}

var altParamIndex int = 0

func getAltSuffix() string {
	altParamIndex++
	return fmt.Sprintf("Alt%d", altParamIndex)
}

func isNewParam(pkgName string, sourceIndex int, param *ast.Field, knownParams []*Field) (*Field, bool) {
	foundParam := convertField(pkgName, sourceIndex, param)
	for _, knownParam := range knownParams {
		if foundParam.Name == knownParam.Name && foundParam.Type == knownParam.Type {
			return foundParam, false
		}
	}
	// if there is a known param that is of a different type, use alt name but include
	for _, knownParam := range knownParams {
		if foundParam.Name == knownParam.Name && foundParam.Type != knownParam.Type {
			foundParam.Name += getAltSuffix()
			return foundParam, true
		}
	}
	return foundParam, true
}

func convertField(pkgName string, sourceIndex int, astField *ast.Field) *Field {
	var field Field
	field.Name = astField.Names[0].Name
	field.SourceIndex = sourceIndex

	typ := typeString("", pkgName, astField.Type)
	if len(typ) == 0 {
		panic(fmt.Sprintf("unhandled field type %s", reflect.TypeOf(astField.Type)))
	}
	field.Type = typ

	return &field
}

func typeString(name string, pkgName string, expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		ident := t.String()
		// resolve exported local types in the package
		if (name == "" || name == "*") && strings.ToUpper(string(ident[0])) == string(ident[0]) && len(pkgName) > 0 {
			return name + pkgName + "." + t.Name
		}
		return name + t.Name

	case *ast.StarExpr:
		return typeString("*", pkgName, t.X)

	case *ast.SelectorExpr:
		return name + typeString("", pkgName, t.X) + "." + t.Sel.Name

	case *ast.ArrayType:
		ret := name + "["
		if t.Len != nil {
			ret += types.ExprString(t.Len)
		}
		return ret + "]" + typeString("", pkgName, t.Elt)

	case *ast.ChanType:
		chanStr := "chan"
		switch t.Dir {
		case ast.SEND:
			chanStr += "<-"
		case ast.RECV:
			chanStr = "<-" + chanStr
		}
		return name + chanStr + " " + typeString("", pkgName, t.Value)

	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeString("", pkgName, t.Key), typeString("", pkgName, t.Value))

	default:
		return name + types.ExprString(expr)
	}
}

func deref(name string) string {
	if name[0] == '*' {
		return name[1:]
	}
	return name
}

func isLocalType(is bool, expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		// if first char is upper case
		if strings.ToUpper(string(t.Name[0])) == string(t.Name[0]) {
			return true && is
		}
		return false

	case *ast.StarExpr:
		return isLocalType(true && is, t.X)

	case *ast.SelectorExpr:
		return false

	default:
		return false
	}
}

func mergeFields(from, to []*Field) []*Field {
	for _, fromField := range from {
		var exists bool
		for _, toField := range to {
			if fromField.Name == toField.Name && fromField.Type == toField.Type {
				exists = true
				break
			}
			if fromField.Name == toField.Name && fromField.Type != toField.Type {
				fromField.Name += getAltSuffix()
				break
			}
		}
		if !exists {
			to = append(to, fromField)
		}
	}
	return to
}

func mergeImport(imp string, imports []string) []string {
	for _, oldImp := range imports {
		if imp == oldImp {
			return imports
		}
	}
	imports = append(imports, imp)
	return imports
}

func hasUnexportedField(fields []*ast.Field) bool {
	for _, field := range fields {
		firstLetter := string(field.Names[0].Name[0])
		if strings.ToLower(firstLetter) == string(firstLetter) {
			return true
		}
	}
	return false
}

func pkgNameToMethodPrefix(pkgName string) string {
	parts := strings.Split(pkgName, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	return strings.Join(parts, "")
}
