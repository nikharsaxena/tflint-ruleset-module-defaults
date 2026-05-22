package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func TestModuleDefaultArgumentRule(t *testing.T) {
	tests := []struct {
		name        string
		calling     string
		modulePath  string // dir under temp root to create, e.g. "modules/foo"
		moduleFiles map[string]string
		wantIssues  int
		wantSubstr  string
	}{
		{
			name: "string default matched — flags",
			calling: `module "x" {
  source = "./modules/foo"
  name   = "default-name"
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `variable "name" { default = "default-name" }`,
			},
			wantIssues: 1,
			wantSubstr: `"name"`,
		},
		{
			name: "string default not matched — clean",
			calling: `module "x" {
  source = "./modules/foo"
  name   = "something-else"
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `variable "name" { default = "default-name" }`,
			},
			wantIssues: 0,
		},
		{
			name: "non-local source — skipped",
			calling: `module "x" {
  source = "hashicorp/consul/aws"
  name   = "anything"
}`,
			wantIssues: 0,
		},
		{
			name: "variable has no default — skipped",
			calling: `module "x" {
  source = "./modules/foo"
  name   = "whatever"
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `variable "name" { type = string }`,
			},
			wantIssues: 0,
		},
		{
			name: "number default matched — flags",
			calling: `module "x" {
  source = "./modules/foo"
  size   = 3
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `variable "size" { default = 3 }`,
			},
			wantIssues: 1,
			wantSubstr: `"size"`,
		},
		{
			name: "bool default matched — flags",
			calling: `module "x" {
  source  = "./modules/foo"
  enabled = true
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `variable "enabled" { default = true }`,
			},
			wantIssues: 1,
		},
		{
			name: "meta-argument count is never flagged",
			calling: `module "x" {
  source = "./modules/foo"
  count  = 1
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				// Even if (improbably) the module declared a var named "count"
				// with default = 1, count is a meta-arg in the *caller* and
				// must never be compared.
				"variables.tf": `variable "count" { default = 1 }`,
			},
			wantIssues: 0,
		},
		{
			name: "multiple args, mixed result",
			calling: `module "x" {
  source  = "./modules/foo"
  name    = "default-name"
  size    = 99
  enabled = true
}`,
			modulePath: "modules/foo",
			moduleFiles: map[string]string{
				"variables.tf": `
					variable "name"    { default = "default-name" }
					variable "size"    { default = 5 }
					variable "enabled" { default = true }
				`,
			},
			wantIssues: 2, // name + enabled match; size doesn't
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()

			if tt.modulePath != "" {
				modDir := filepath.Join(tmp, tt.modulePath)
				if err := os.MkdirAll(modDir, 0o755); err != nil {
					t.Fatal(err)
				}
				for name, content := range tt.moduleFiles {
					p := filepath.Join(modDir, name)
					if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}

			// chdir so that filepath.Dir("main.tf") == "." resolves under tmp.
			// The rule reads the child module from disk, so its working dir
			// must contain the modules/ subtree we just created.
			oldwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(tmp); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(oldwd) })

			rule := NewModuleDefaultArgumentRule()
			runner := helper.TestRunner(t, map[string]string{
				"main.tf": tt.calling,
			})

			if err := rule.Check(runner); err != nil {
				t.Fatalf("rule.Check: %v", err)
			}

			if got := len(runner.Issues); got != tt.wantIssues {
				t.Fatalf("got %d issues, want %d; issues=%+v", got, tt.wantIssues, runner.Issues)
			}

			if tt.wantIssues > 0 && tt.wantSubstr != "" {
				if !strings.Contains(runner.Issues[0].Message, tt.wantSubstr) {
					t.Errorf("first issue %q lacks substring %q", runner.Issues[0].Message, tt.wantSubstr)
				}
			}
		})
	}
}
