# Troubleshooting

Common issues and how to resolve them.

## Configuration

### Config file not found

```text
Error: configuration file not found: .terratidy.yaml (run 'terratidy init' to create one)
```

TerraTidy looks for `.terratidy.yaml` in the current directory. Run `terratidy init` to create one,
or specify a path with `--config`.

### Invalid config syntax

```text
Error: loading config: parsing YAML: ...
```

Run `terratidy config validate` to check your configuration file for syntax errors, invalid
values, or missing required fields.

### No engines enabled

```text
Warning: no engines are enabled
```

At least one engine must be enabled. Check your config or profile to ensure `enabled: true`
is set for at least one engine.

## File Discovery

### No HCL files found

```text
No HCL files found
```

TerraTidy processes `.tf`, `.hcl`, and `.tfvars` files. Check that:

- You're running from the correct directory
- Your files have the correct extension
- Files aren't in a skipped directory (node_modules, vendor, .terraform, .terragrunt-cache)

### --changed requires git

```text
Error: not a git repository; --changed requires git
```

The `--changed` flag uses git to detect modified files. Run from within a git repository,
or omit the flag to check all files.

## Plugin Issues

### Unsupported rule type

```text
Error: unsupported rule type: bash (use go, rego, or yaml)
```

The `init-rule` command supports `go`, `rego`, and `yaml` types. Bash rules must be
created manually.

### Bash rule not executable

```text
Error: plugin script is not executable
```

Bash rules must be executable. Fix with:

```bash
chmod +x .terratidy/plugins/my-rule.sh
```

### Plugin loading errors

Check that:

- Plugin directories exist and are readable
- Go plugins (`.so`) are compiled for the correct platform
- YAML rules have valid syntax
- Bash scripts produce valid JSON output

## LSP Issues

### Server not starting

```bash
# Test if server starts
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | terratidy lsp
```

Check that TerraTidy is in your PATH and the configuration file is valid.

### No diagnostics appearing

- Ensure the file is saved (diagnostics trigger on save)
- Check engine configuration in `.terratidy.yaml`
- Verify severity threshold isn't filtering findings
- Test the LSP server manually: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | terratidy lsp`

## Performance

### Slow on large repositories

- Use `--changed` to only check modified files
- Enable `--parallel` for the check command
- Disable engines you don't need (`--skip-lint`, `--skip-policy`)
- Use a profile with fewer engines for local development
- The cache (5-minute TTL, 1000 entries) helps with repeated runs

### TFLint not found

```text
Error: tflint not found in PATH (install: brew install tflint, or see https://github.com/terraform-linters/tflint#installation)
```

The lint engine falls back to built-in rules when TFLint is not installed. To use TFLint:

```bash
# Install TFLint
brew install tflint  # macOS
# or
curl -s https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash
```

## Exit Codes

| Code | Meaning | Action |
|------|---------|--------|
| `0`  | Success, no findings | All good |
| `1`  | Findings found | Run `terratidy fix` or review findings |
| `2`  | Configuration error | Check `.terratidy.yaml` syntax and required fields |
| `3`  | Internal error | Check permissions, disk space, or report a bug |

**Exit code 1 (findings found):** Check the severity of your findings. Only error-severity
findings cause exit code 1. Use `--severity-threshold error` to ignore warnings and info.

**Exit code 2 (configuration error):** Common causes:

- Malformed YAML in `.terratidy.yaml`
- Invalid profile name
- Plugin loading failure (missing or incompatible plugin)
- Missing required configuration fields

**Exit code 3 (internal error):** These are unexpected failures. Check:

- File permissions
- Disk space
- Report persistent issues at <https://github.com/santosr2/TerraTidy/issues>
