# tflint-ruleset-module-defaults

A [tflint](https://github.com/terraform-linters/tflint) ruleset that flags
**redundant module arguments** — places where a caller sets a variable to the
exact value the module already declares as its default.

```hcl
# In your module:
variable "description" {
  type    = string
  default = ""
}

# In a caller — this rule will warn:
module "x" {
  source      = "./modules/group"
  description = ""   # redundant; matches the default
}
```

## Why

Setting a module argument to its declared default is noise: it makes call sites
longer, suggests intent that doesn't exist, and silently de-syncs from the
module if the default ever changes. This rule catches it mechanically so reviewers
don't have to cross-reference `variables.tf` every time.

## Install

Add to your `.tflint.hcl`:

```hcl
plugin "module-defaults" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/nikharsaxena/tflint-ruleset-module-defaults"
}
```

Then run `tflint --init` to download the plugin.

## Rules

| Name                       | Default severity | Default enabled |
|----------------------------|------------------|-----------------|
| `module_default_argument`  | `WARNING`        | yes             |

## Scope

**v0.1 supports local module sources only** — `./path`, `../path`, `/abs/path`.
Registry, git, and other remote sources are skipped silently. The reason: remote
modules need fetching, version resolution, and caching, which adds substantial
complexity and failure modes. Local modules cover the common case for monorepos.

The rule compares values structurally using cty's `RawEquals`, so it handles
strings, numbers, bools, lists, and objects/maps — including empty-collection
defaults like `default = {}`.

## Limitations

* **Expression-level comparison.** If a caller writes
  `description = local.empty_string` and `local.empty_string == ""`, the rule
  evaluates both sides; if both fold to the same constant in an empty context,
  it flags the override. References that the empty context can't resolve are
  skipped silently rather than producing false positives.
* **Defaults referencing other constructs are skipped.** If a module's default
  is itself a function call or variable reference (e.g. `default = upper("x")`),
  the rule cannot establish a confident equality and moves on.
* **Meta-arguments are never flagged.** `count`, `for_each`, `depends_on`,
  `providers`, `source`, and `version` are reserved on the caller side and are
  excluded from comparison even if a child module declares a `variable` block
  with the same name.

## Building from source

```sh
go build -o tflint-ruleset-module-defaults ./cmd/
mkdir -p ~/.tflint.d/plugins
mv tflint-ruleset-module-defaults ~/.tflint.d/plugins/
```

Then in `.tflint.hcl`, declare the plugin block without `source`/`version` to
use the locally-installed binary:

```hcl
plugin "module-defaults" {
  enabled = true
}
```

## Tests

```sh
go test ./...
```

## License

MPL-2.0
