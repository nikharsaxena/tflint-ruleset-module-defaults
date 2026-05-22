package rules

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestLoadModuleDefaults(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		wantKeys   []string
		wantAbsent []string
	}{
		{
			name: "single variable with string default",
			files: map[string]string{
				"variables.tf": `variable "foo" { default = "bar" }`,
			},
			wantKeys: []string{"foo"},
		},
		{
			name: "variable without default is excluded",
			files: map[string]string{
				"variables.tf": `variable "no_default" { type = string }`,
			},
			wantAbsent: []string{"no_default"},
		},
		{
			name: "multiple files merged",
			files: map[string]string{
				"variables.tf":      `variable "a" { default = 1 }`,
				"more_variables.tf": `variable "b" { default = true }`,
			},
			wantKeys: []string{"a", "b"},
		},
		{
			name: "non-tf files ignored",
			files: map[string]string{
				"variables.tf": `variable "foo" { default = "bar" }`,
				"README.md":    `# not terraform`,
			},
			wantKeys: []string{"foo"},
		},
		{
			name: "mixed defaulted and non-defaulted in one file",
			files: map[string]string{
				"variables.tf": `
					variable "with_default"    { default = "x" }
					variable "without_default" { type = string }
				`,
			},
			wantKeys:   []string{"with_default"},
			wantAbsent: []string{"without_default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				path := filepath.Join(dir, name)
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %q: %v", name, err)
				}
			}

			got, err := loadModuleDefaults(dir)
			if err != nil {
				t.Fatalf("loadModuleDefaults: %v", err)
			}

			gotKeys := keysOf(got)
			sort.Strings(gotKeys)
			for _, k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Errorf("missing expected key %q (got keys: %v)", k, gotKeys)
				}
			}
			for _, k := range tt.wantAbsent {
				if _, ok := got[k]; ok {
					t.Errorf("unexpected key %q present", k)
				}
			}
		})
	}
}

func TestLoadModuleDefaults_MissingDir(t *testing.T) {
	_, err := loadModuleDefaults(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing dir, got nil")
	}
}

func TestIsLocalSource(t *testing.T) {
	cases := map[string]bool{
		"./modules/foo":           true,
		"../shared/bar":           true,
		"/absolute/path":          true,
		"modules/foo":             false, // bare path — Terraform treats as registry
		"hashicorp/consul/aws":    false,
		"git::https://example.git": false,
		"github.com/foo/bar":      false,
	}
	for src, want := range cases {
		if got := isLocalSource(src); got != want {
			t.Errorf("isLocalSource(%q) = %v, want %v", src, got, want)
		}
	}
}

func keysOf[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
