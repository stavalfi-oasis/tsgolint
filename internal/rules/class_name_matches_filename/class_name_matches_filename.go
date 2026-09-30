package class_name_matches_filename

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

var extension = regexp.MustCompile(`\.[cm]?[jt]sx?$`)

func buildMismatchMessage(name string, expected string, base string) rule.RuleMessage {
	return rule.RuleMessage{
		Id:          "classNameMismatch",
		Description: "Class '" + name + "' must be named '" + expected + "' to match the file '" + base + "'.",
		Help:        "Rename the class, or move it to '" + asFileName(name) + ".ts', so an import site names the file it comes from.",
	}
}

// "gh-gateway.service.ts" -> "GhGatewayService".
func expectedName(base string) string {
	stem := extension.ReplaceAllString(base, "")
	if stem == "" || stem == "index" {
		return ""
	}
	var built strings.Builder
	for _, part := range strings.FieldsFunc(stem, func(r rune) bool { return r == '-' || r == '.' }) {
		runes := []rune(part)
		built.WriteRune(unicode.ToUpper(runes[0]))
		built.WriteString(string(runes[1:]))
	}
	return built.String()
}

// "GhGatewayService" -> "gh-gateway-service", the file this class would fit.
func asFileName(name string) string {
	runes := []rune(name)
	var built strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			previous := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextIsLower) {
				built.WriteRune('-')
			}
		}
		built.WriteRune(unicode.ToLower(r))
	}
	return built.String()
}

var ClassNameMatchesFilenameRule = rule.Rule{
	Name: "class-name-matches-filename",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		base := filepath.Base(ctx.SourceFile.FileName())
		expected := expectedName(base)
		if expected == "" {
			return rule.RuleListeners{}
		}

		return rule.RuleListeners{
			ast.KindClassDeclaration: func(node *ast.Node) {
				// Only the file's own top-level class is pinned to the filename;
				// a class nested inside a function or block is a local detail.
				if node.Parent == nil || !ast.IsSourceFile(node.Parent) {
					return
				}
				name := node.Name()
				if name == nil || name.Text() == expected {
					return
				}

				ctx.ReportNodeWithFixes(node, buildMismatchMessage(name.Text(), expected, base), func() []rule.RuleFix {
					// Renaming by resolved symbol rather than by identifier text is
					// what keeps a same-named local in another scope untouched.
					symbol := ctx.TypeChecker.GetSymbolAtLocation(name)
					if symbol == nil {
						return []rule.RuleFix{rule.RuleFixReplace(ctx.SourceFile, name, expected)}
					}

					starts := []int{}
					fixes := []rule.RuleFix{}
					var walk func(current *ast.Node) bool
					walk = func(current *ast.Node) bool {
						if ast.IsIdentifier(current) && current.Text() == name.Text() &&
							ctx.TypeChecker.GetSymbolAtLocation(current) == symbol &&
							!slices.Contains(starts, current.Pos()) {
							starts = append(starts, current.Pos())
							fixes = append(fixes, rule.RuleFixReplace(ctx.SourceFile, current, expected))
						}
						ast.ForEachChildAndJSDoc(current, ctx.SourceFile, walk)
						return false
					}
					walk(ctx.SourceFile.AsNode())
					return fixes
				})
			},
		}
	},
}
