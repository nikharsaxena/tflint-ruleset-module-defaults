// Command tflint-ruleset-module-defaults is the plugin entrypoint.
//
// tflint loads this binary via the go-plugin protocol when the ruleset is
// listed in .tflint.hcl. The Serve call wires our rules into tflint's runner.
package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/nikharsaxena/tflint-ruleset-module-defaults/rules"
)

// version is overridden at build time via `-ldflags="-X main.version=…"`.
var version = "0.1.0"

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "module-defaults",
			Version: version,
			Rules: []tflint.Rule{
				rules.NewModuleDefaultArgumentRule(),
			},
		},
	})
}
