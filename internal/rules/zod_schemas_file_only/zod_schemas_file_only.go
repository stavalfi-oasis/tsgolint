package zod_schemas_file_only

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const schemasFileName = "zod-schemas.ts"

func buildWrongFileMessage(name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "schemaOutsideSchemasFile",
		Description: "Zod schema '" + name + "' is declared outside " + schemasFileName + ".",
		Help:        "Move it to the package's " + schemasFileName + " and import it, so every schema in a package has one home.",
	}
}

func buildDuplicateFileMessage(packageDir string, other string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "duplicateSchemasFile",
		Description: "Package '" + packageDir + "' has more than one " + schemasFileName + "; also found at '" + other + "'.",
		Help:        "A package is the directory of its tsconfig.json, and may hold exactly one " + schemasFileName + ". Merge them.",
	}
}

// A package is the nearest ancestor directory holding a tsconfig.json. Returns
// "" when there is none, which leaves the file unattributed and unchecked for
// duplicates.
func packageDirOf(fileName string) string {
	dir := filepath.Dir(fileName)
	for {
		if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// zod's runtime classes are all named ZodSomething (ZodObject, ZodString, ...),
// so the declared type identifies a schema regardless of how it was built or
// what the binding is called.
func isZodSchemaType(typeChecker *checker.Checker, node *ast.Node) bool {
	t := utils.GetConstrainedTypeAtLocation(typeChecker, node)
	return utils.SomeUnionTypePart(t, func(part *checker.Type) bool {
		symbol := checker.Type_symbol(part)
		return symbol != nil && strings.HasPrefix(symbol.Name, "Zod")
	})
}

var ZodSchemasFileOnlyRule = rule.Rule{
	Name: "zod-schemas-file-only",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		fileName := ctx.SourceFile.FileName()
		isSchemasFile := filepath.Base(fileName) == schemasFileName

		listeners := rule.RuleListeners{}

		if !isSchemasFile {
			listeners[ast.KindVariableDeclaration] = func(node *ast.Node) {
				// `declare const schema: ZodString` introduces no schema — it
				// describes one that lives elsewhere, same as an import.
				if node.Flags&ast.NodeFlagsAmbient != 0 {
					return
				}

				declaration := node.AsVariableDeclaration()
				name := declaration.Name()
				if name == nil || !ast.IsIdentifier(name) {
					return
				}
				// Check the binding, not the initialiser: a schema assigned from
				// a helper or a function call is still a schema living here.
				if !isZodSchemaType(ctx.TypeChecker, name) {
					return
				}
				ctx.ReportNode(name, buildWrongFileMessage(name.Text()))
			}
			return listeners
		}

		// This file is a zod-schemas.ts. Report once, on the file, if its package
		// holds another one. Every zod-schemas.ts in the package reports, so the
		// diagnostic is visible whichever file you happen to be linting.
		listeners[ast.KindSourceFile] = func(node *ast.Node) {
			packageDir := packageDirOf(fileName)
			if packageDir == "" {
				return
			}

			siblings := []string{}
			for _, sourceFile := range ctx.Program.SourceFiles() {
				other := sourceFile.FileName()
				if other == fileName || filepath.Base(other) != schemasFileName {
					continue
				}
				if packageDirOf(other) == packageDir {
					siblings = append(siblings, other)
				}
			}
			if len(siblings) == 0 {
				return
			}
			sort.Strings(siblings)

			ctx.ReportRange(
				ctx.SourceFile.Loc.WithEnd(ctx.SourceFile.Loc.Pos()),
				buildDuplicateFileMessage(packageDir, siblings[0]),
			)
		}

		return listeners
	},
}
