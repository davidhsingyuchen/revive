package rule

import (
	"fmt"
	"github.com/fatih/structtag"
	"github.com/mgechev/revive/lint"
	"go/ast"
	"strconv"
	"strings"
)

// StructTagRule lints struct tags.
type StructTagRule struct{}

// Apply applies the rule to given file.
func (r *StructTagRule) Apply(file *lint.File, _ lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	w := lintStructTagRule{onFailure: onFailure}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (r *StructTagRule) Name() string {
	return "struct-tag"
}

type lintStructTagRule struct {
	onFailure func(lint.Failure)
}

func (w lintStructTagRule) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.StructType:
		if n.Fields == nil || n.Fields.NumFields() < 1 {
			return nil // skip empty structs
		}

		for _, f := range n.Fields.List {
			if f.Tag != nil {
				w.checkTaggedField(f)
			}
		}
	}

	return w

}

// checkTaggedField checks the tag of the given field.
// precondition: the field has a tag
func (w lintStructTagRule) checkTaggedField(f *ast.Field) {
	tags, err := structtag.Parse(strings.Trim(f.Tag.Value, "`"))
	if err != nil || tags == nil {
		return
	}

	for _, tag := range tags.Tags() {
		switch key := tag.Key; key {
		case "asn1":
			// Not implemented yet
		case "default":
			if !w.typeValueMatch(f.Type, tag.Name) {
				w.addFailure(f.Tag, "field's type and default value's type mismatch")
			}
		case "json":
			msg, ok := w.checkJSONTag(tag.Options)
			if !ok {
				w.addFailure(f.Tag, msg)
			}
		case "protobuf":
			// Not implemented yet
		case "required":
			if tag.Name != "true" && tag.Name != "false" {
				w.addFailure(f.Tag, "required should be 'true' or 'false'")
			}
		case "xml":
			msg, ok := w.checkXMLTag(tag.Options)
			if !ok {
				w.addFailure(f.Tag, msg)
			}
		case "yaml":
			msg, ok := w.checkYAMLTag(tag.Options)
			if !ok {
				w.addFailure(f.Tag, msg)
			}
		default:
			// unknown key
		}
	}
}

func (w lintStructTagRule) checkJSONTag(options []string) (string, bool) {
	for _, opt := range options {
		switch opt {
		case "omitempty", "string":
		default:
			return fmt.Sprintf("unknown option '%s' in JSON tag", opt), false
		}
	}

	return "", true
}

func (w lintStructTagRule) checkXMLTag(options []string) (string, bool) {
	for _, opt := range options {
		switch opt {
		case "attr", "cdata", "chardata", "innerxml", "comment", "any", "omitempty":
		default:
			return fmt.Sprintf("unknown option '%s' in XML tag", opt), false
		}
	}

	return "", true
}

func (w lintStructTagRule) checkYAMLTag(options []string) (string, bool) {
	for _, opt := range options {
		switch opt {
		case "flow", "inline", "omitempty":
		default:
			return fmt.Sprintf("unknown option '%s' in YAML tag", opt), false
		}
	}

	return "", true
}

func (w lintStructTagRule) typeValueMatch(t ast.Expr, val string) bool {
	tID, ok := t.(*ast.Ident)
	if !ok {
		return true
	}

	typeMatches := true
	switch tID.Name {
	case "bool":
		typeMatches = val == "true" || val == "false"
	case "float64":
		_, err := strconv.ParseFloat(val, 64)
		typeMatches = err == nil
	case "int":
		_, err := strconv.ParseInt(val, 10, 64)
		typeMatches = err == nil
	case "string":
	case "nil":
	default:
		// unchecked type
	}

	return typeMatches
}

func (w lintStructTagRule) addFailure(n ast.Node, msg string) {
	w.onFailure(lint.Failure{
		Node:       n,
		Failure:    msg,
		Confidence: 1,
	})
}