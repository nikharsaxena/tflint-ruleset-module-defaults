package rules

import (
	"fmt"
	"path/filepath"

	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/zclconf/go-cty/cty"
)

// ModuleDefaultArgumentRule warns when a module block sets an argument to a
// value that equals the module's declared default — i.e. the assignment is
// redundant and can be removed without changing behaviour.
type ModuleDefaultArgumentRule struct {
	tflint.DefaultRule
}

func NewModuleDefaultArgumentRule() *ModuleDefaultArgumentRule {
	return &ModuleDefaultArgumentRule{}
}

func (r *ModuleDefaultArgumentRule) Name() string         { return "module_default_argument" }
func (r *ModuleDefaultArgumentRule) Enabled() bool        { return true }
func (r *ModuleDefaultArgumentRule) Severity() tflint.Severity { return tflint.WARNING }
func (r *ModuleDefaultArgumentRule) Link() string         { return "" }

// metaArgs are Terraform's reserved module-block meta-arguments. They are never
// declared as `variable` blocks inside a module, so the rule should never
// compare them against module defaults.
var metaArgs = map[string]bool{
	"source":     true,
	"version":    true,
	"count":      true,
	"for_each":   true,
	"providers":  true,
	"depends_on": true,
}

func (r *ModuleDefaultArgumentRule) Check(runner tflint.Runner) error {
	body, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "module",
				LabelNames: []string{"name"},
				Body: &hclext.BodySchema{
					Mode: hclext.SchemaJustAttributesMode,
				},
			},
		},
	}, nil)
	if err != nil {
		return err
	}

	for _, modBlock := range body.Blocks {
		sourceAttr, ok := modBlock.Body.Attributes["source"]
		if !ok {
			continue
		}

		var source string
		if err := runner.EvaluateExpr(sourceAttr.Expr, &source, nil); err != nil {
			continue
		}

		if !isLocalSource(source) {
			continue
		}

		callerDir := filepath.Dir(sourceAttr.Range.Filename)
		modulePath := filepath.Join(callerDir, source)

		defaults, err := loadModuleDefaults(modulePath)
		if err != nil {
			// Module directory unreadable, missing, or has parse errors —
			// nothing we can compare against, so skip silently rather than
			// failing the whole tflint run.
			continue
		}

		for name, attr := range modBlock.Body.Attributes {
			if metaArgs[name] {
				continue
			}
			defaultExpr, hasDefault := defaults[name]
			if !hasDefault {
				continue
			}

			defaultVal, defaultDiags := defaultExpr.Value(nil)
			if defaultDiags.HasErrors() {
				// Default references something we can't resolve (a function
				// call, another var, etc.). Skip — we can't make a confident
				// equality claim.
				continue
			}

			var callerVal cty.Value
			if err := runner.EvaluateExpr(attr.Expr, &callerVal, nil); err != nil {
				continue
			}

			if callerVal.RawEquals(defaultVal) {
				if err := runner.EmitIssue(
					r,
					fmt.Sprintf("argument %q is set to the module's default value", name),
					attr.Range,
				); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
