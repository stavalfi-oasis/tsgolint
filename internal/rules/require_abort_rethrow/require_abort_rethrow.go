package require_abort_rethrow

import (
	"path/filepath"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type RequireAbortRethrowOptions struct {
	// A file, relative to the working directory, imported by a relative path
	// from the linted file. Wins over ImportSource, for the package that owns
	// the guard and cannot import itself by name.
	ImportFile   string `json:"importFile"`
	ImportSource string `json:"importSource"`
}

const (
	guardName           = "isAbortError"
	defaultImportSource = "@poc/libs/utils.ts"
	defaultBindingName  = "error"
)

func buildMissingAbortRethrowMessage(guard string, name string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "missingAbortRethrow",
		Description: "The first statement of a catch block must rethrow abort errors, or an abort is silently swallowed.",
		Help:        "Start the block with `if (" + guard + "(" + name + ")) { throw " + name + "; }`.",
	}
}

func isIdentifierNamed(node *ast.Node, name string) bool {
	return node != nil && ast.IsIdentifier(node) && node.Text() == name
}

func isThrowOf(node *ast.Node, name string) bool {
	if node == nil {
		return false
	}
	if node.Kind == ast.KindBlock {
		statements := node.AsBlock().Statements.Nodes
		return len(statements) == 1 && isThrowOf(statements[0], name)
	}
	return node.Kind == ast.KindThrowStatement && isIdentifierNamed(node.AsThrowStatement().Expression, name)
}

func isGuardCall(node *ast.Node, guard string, name string) bool {
	if node == nil || !ast.IsCallExpression(node) {
		return false
	}
	call := node.AsCallExpression()
	return isIdentifierNamed(call.Expression, guard) &&
		call.Arguments != nil &&
		len(call.Arguments.Nodes) == 1 &&
		isIdentifierNamed(call.Arguments.Nodes[0], name)
}

// `if (isAbortError(name)) { throw name; }` — with or without the braces, and
// no else branch, since that could still handle the abort. A first statement
// testing `!isAbortError(name)` is a deliberate decision about aborts too
// (shutdown loops swallow them on purpose), so it also counts.
func isAbortHandled(statement *ast.Node, guard string, name string) bool {
	if statement.Kind != ast.KindIfStatement {
		return false
	}
	ifStatement := statement.AsIfStatement()
	condition := ifStatement.Expression
	if condition.Kind == ast.KindPrefixUnaryExpression {
		unary := condition.AsPrefixUnaryExpression()
		return unary.Operator == ast.KindExclamationToken && isGuardCall(unary.Operand, guard, name)
	}
	return ifStatement.ElseStatement == nil &&
		isGuardCall(condition, guard, name) &&
		isThrowOf(ifStatement.ThenStatement, name)
}

// The whitespace a line starts with, for the line holding `position`.
func lineIndent(text string, position int) string {
	start := strings.LastIndexByte(text[:position], '\n') + 1
	end := start
	for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
		end++
	}
	return text[start:end]
}

func relativeSpecifier(fromFile string, importFile string) (string, bool) {
	target, err := filepath.Abs(importFile)
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(filepath.Dir(fromFile), target)
	if err != nil {
		return "", false
	}
	relative = filepath.ToSlash(relative)
	if !strings.HasPrefix(relative, "../") {
		relative = "./" + relative
	}
	return relative, true
}

var RequireAbortRethrowRule = rule.Rule{
	Name: "require-abort-rethrow",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		opts := utils.UnmarshalOptions[RequireAbortRethrowOptions](options, "require-abort-rethrow")
		importSource := opts.ImportSource
		if importSource == "" {
			importSource = defaultImportSource
		}
		if opts.ImportFile != "" {
			if relative, ok := relativeSpecifier(ctx.SourceFile.FileName(), opts.ImportFile); ok {
				importSource = relative
			}
		}

		text := ctx.SourceFile.Text()
		guard := guardName
		guardInScope := false
		importFixEmitted := false
		var lastImport *ast.Node
		var firstStatement *ast.Node

		// One import fix per file, and none when the guard is already imported
		// or declared here.
		importFix := func() []rule.RuleFix {
			if guardInScope || importFixEmitted {
				return nil
			}
			importFixEmitted = true
			line := "import { " + guardName + " } from \"" + importSource + "\";"
			if lastImport != nil {
				return []rule.RuleFix{rule.RuleFixInsertAfter(lastImport, "\n"+line)}
			}
			if firstStatement != nil {
				return []rule.RuleFix{rule.RuleFixInsertBefore(ctx.SourceFile, firstStatement, line+"\n")}
			}
			return nil
		}

		return rule.RuleListeners{
			ast.KindSourceFile: func(node *ast.Node) {
				statements := node.AsSourceFile().Statements.Nodes
				if len(statements) > 0 {
					firstStatement = statements[0]
				}
				for _, statement := range statements {
					switch statement.Kind {
					case ast.KindImportDeclaration:
						lastImport = statement
						clause := statement.AsImportDeclaration().ImportClause
						if clause == nil {
							continue
						}
						bindings := clause.AsImportClause().NamedBindings
						if bindings == nil || bindings.Kind != ast.KindNamedImports {
							continue
						}
						for _, element := range bindings.AsNamedImports().Elements.Nodes {
							specifier := element.AsImportSpecifier()
							imported := specifier.PropertyName
							if imported == nil {
								imported = specifier.Name()
							}
							if imported != nil && imported.Text() == guardName {
								guardInScope = true
								guard = specifier.Name().Text()
							}
						}
					case ast.KindFunctionDeclaration:
						if isIdentifierNamed(statement.Name(), guardName) {
							guardInScope = true
						}
					case ast.KindVariableStatement:
						for _, declaration := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
							if isIdentifierNamed(declaration.Name(), guardName) {
								guardInScope = true
							}
						}
					}
				}
			},

			ast.KindCatchClause: func(node *ast.Node) {
				clause := node.AsCatchClause()
				block := clause.Block
				statements := block.AsBlock().Statements.Nodes

				name := defaultBindingName
				if clause.VariableDeclaration != nil {
					binding := clause.VariableDeclaration.Name()
					if !ast.IsIdentifier(binding) {
						// A destructured binding has no single value to rethrow.
						ctx.ReportNode(node, buildMissingAbortRethrowMessage(guard, defaultBindingName))
						return
					}
					name = binding.Text()
				}

				if len(statements) > 0 && isAbortHandled(statements[0], guard, name) {
					return
				}

				ctx.ReportNodeWithFixes(node, buildMissingAbortRethrowMessage(guard, name), func() []rule.RuleFix {
					catchStart := utils.TrimNodeTextRange(ctx.SourceFile, node).Pos()
					braceStart := utils.TrimNodeTextRange(ctx.SourceFile, block).Pos()
					catchIndent := lineIndent(text, catchStart)
					indent := catchIndent + "  "
					// Where the existing body starts on a line of its own, the guard
					// takes that line's indentation; otherwise the body is pushed down.
					trailer := "\n" + catchIndent
					replaceEnd := braceStart + 1
					if len(statements) > 0 {
						firstStart := utils.TrimNodeTextRange(ctx.SourceFile, statements[0]).Pos()
						trailer = "\n" + indent
						replaceEnd = firstStart
						if strings.Contains(text[braceStart:firstStart], "\n") {
							indent = lineIndent(text, firstStart)
							trailer = ""
							replaceEnd = braceStart + 1
						}
					} else if strings.Contains(text[braceStart:block.End()], "\n") {
						trailer = ""
					}

					guardText := "\n" + indent + "if (" + guard + "(" + name + ")) {\n" +
						indent + "  throw " + name + ";\n" +
						indent + "}" + trailer

					fixes := []rule.RuleFix{}
					if clause.VariableDeclaration == nil {
						fixes = append(fixes, rule.RuleFixReplaceRange(
							core.NewTextRange(catchStart+len("catch"), braceStart),
							" ("+name+") ",
						))
					}
					fixes = append(fixes, rule.RuleFixReplaceRange(
						core.NewTextRange(braceStart+1, replaceEnd),
						guardText,
					))
					return append(fixes, importFix()...)
				})
			},
		}
	},
}
