// Package rules implements the tflint rules for this ruleset.
package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// loadModuleDefaults walks every *.tf file directly inside dir, finds every
// `variable "x" { default = ... }` block, and returns name → default expression.
// Variables without a default are omitted. Subdirectories are not descended.
func loadModuleDefaults(dir string) (map[string]hcl.Expression, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read module dir %q: %w", dir, err)
	}

	parser := hclparse.NewParser()
	defaults := map[string]hcl.Expression{}

	fileSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "variable", LabelNames: []string{"name"}},
		},
	}
	varSchema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "default"}},
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
		file, diags := parser.ParseHCL(src, path)
		if diags.HasErrors() {
			return nil, fmt.Errorf("parse %q: %s", path, diags.Error())
		}
		content, _, diags := file.Body.PartialContent(fileSchema)
		if diags.HasErrors() {
			return nil, fmt.Errorf("decode %q: %s", path, diags.Error())
		}
		for _, block := range content.Blocks {
			name := block.Labels[0]
			varContent, _, varDiags := block.Body.PartialContent(varSchema)
			if varDiags.HasErrors() {
				continue
			}
			if defaultAttr, ok := varContent.Attributes["default"]; ok {
				defaults[name] = defaultAttr.Expr
			}
		}
	}

	return defaults, nil
}

// isLocalSource reports whether a module source string refers to a local path
// (relative `./`, `../`, or absolute). Registry, git, and other remote sources
// return false — those need fetching/cloning and are out of scope for v0.1.
func isLocalSource(source string) bool {
	return strings.HasPrefix(source, "./") ||
		strings.HasPrefix(source, "../") ||
		strings.HasPrefix(source, "/")
}
