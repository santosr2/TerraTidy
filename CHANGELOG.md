# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### CI/CD

- **release**: Re-release existing tags and bump goreleaser to v2.16.0 ([#250](https://github.com/santosr2/TerraTidy/pull/250)) by [@santosr2](https://github.com/santosr2)
- **release**: Fix downstream release jobs and stop alias-tag loop ([#251](https://github.com/santosr2/TerraTidy/pull/251)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog for v0.2.0 ([ffe738c](https://github.com/santosr2/TerraTidy/commit/ffe738c57636dbf03d4ba8f3f915e368752e4e93)) by [@github-actions[bot]](https://github.com/github-actions[bot])

## [0.2.0] - 2026-07-22

### Added

- **plugins**: Integrate plugin rules into style and lint engines ([#93](https://github.com/santosr2/TerraTidy/pull/93)) by [@santosr2](https://github.com/santosr2)
- **plugins**: Add block_types, forbidden_attributes, and attribute_patterns to YAML rules ([#94](https://github.com/santosr2/TerraTidy/pull/94)) by [@santosr2](https://github.com/santosr2)
- Add cache config and LSP engine toggles ([#95](https://github.com/santosr2/TerraTidy/pull/95)) by [@santosr2](https://github.com/santosr2)
- **cli**: Add --exclude flag for glob-based file filtering ([#98](https://github.com/santosr2/TerraTidy/pull/98)) by [@santosr2](https://github.com/santosr2)
- **engines**: Add suppression annotation support ([#99](https://github.com/santosr2/TerraTidy/pull/99)) by [@santosr2](https://github.com/santosr2)
- **cli**: Add --no-recurse flag for non-recursive scanning ([#100](https://github.com/santosr2/TerraTidy/pull/100)) by [@santosr2](https://github.com/santosr2)
- **cli**: Add --absolute-paths flag for absolute file paths in output ([#101](https://github.com/santosr2/TerraTidy/pull/101)) by [@santosr2](https://github.com/santosr2)
- **lsp**: Add debouncing, config watching, and comprehensive tests ([d021dc9](https://github.com/santosr2/TerraTidy/commit/d021dc95abcefbaa915f197aa2eafcf7aa9abe6b)) by [@santosr2](https://github.com/santosr2)
- **cli**: Improve fmt command config handling and add tests ([2af065d](https://github.com/santosr2/TerraTidy/commit/2af065d41fd8794a3202d56bde2e9a4a84aab310)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Add exit codes, config redesign, and CLI integration ([5106349](https://github.com/santosr2/TerraTidy/commit/5106349f2c5fef56f8ed28a32eb9617cf425c00a)) by [@santosr2](https://github.com/santosr2)
- Cli UX improvements and config redesign ([#120](https://github.com/santosr2/TerraTidy/pull/120)) by [@santosr2](https://github.com/santosr2)
- **style**: Detect fix loops via content hashing ([#143](https://github.com/santosr2/TerraTidy/pull/143)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Add Finding.IsDiff for diff-as-message routing ([#144](https://github.com/santosr2/TerraTidy/pull/144)) by [@santosr2](https://github.com/santosr2)
- **engines**: Preserve original file permissions on fmt and style writes ([#145](https://github.com/santosr2/TerraTidy/pull/145)) by [@santosr2](https://github.com/santosr2)
- **cli**: Add --no-parallel flag to check command ([#146](https://github.com/santosr2/TerraTidy/pull/146)) by [@santosr2](https://github.com/santosr2)
- **fmt**: Structured output mode + diff-color polish ([#153](https://github.com/santosr2/TerraTidy/pull/153)) by [@santosr2](https://github.com/santosr2)
- **fmt**: Indent --check --diff output and add unformatted-file summary ([#158](https://github.com/santosr2/TerraTidy/pull/158)) by [@santosr2](https://github.com/santosr2)
- **plugins**: Scaffold .gitignore and tidy target in plugins init ([f7cea97](https://github.com/santosr2/TerraTidy/commit/f7cea97ee9535cb5f6ff3aaa94156632e15f37ac)) by [@santosr2](https://github.com/santosr2)
- **cst**: Add concrete syntax tree package for structural fixes ([#183](https://github.com/santosr2/TerraTidy/pull/183)) by [@santosr2](https://github.com/santosr2)
- **style**: Add terragrunt-include-first rule on top of CST ([#197](https://github.com/santosr2/TerraTidy/pull/197)) by [@santosr2](https://github.com/santosr2)

### Breaking Changes

- Clarify Fixer tie-break, WholeFileEdit caveat, breaking-change parsers ([a1d5a80](https://github.com/santosr2/TerraTidy/commit/a1d5a808ed01632e04a875ad6d18f1c5c189d064)) by [@santosr2](https://github.com/santosr2)

### CI/CD

- Expand workflow coverage with container and precommit tests ([7e82de2](https://github.com/santosr2/TerraTidy/commit/7e82de2abd8b1162190fe28d8f0045ad47115035)) by [@santosr2](https://github.com/santosr2)
- Fix workflow failures in container tests and pre-commit ([298577b](https://github.com/santosr2/TerraTidy/commit/298577b2cf96201c8de57ea0bf81e0caa1729c81)) by [@santosr2](https://github.com/santosr2)
- **fuzz**: Make fuzz workflow label-triggered with PR comments ([d8d5a99](https://github.com/santosr2/TerraTidy/commit/d8d5a990ad4a111bee9d2bbae01c9a6af4fc93d2)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Allow Dependabot PRs to regenerate bun.lockb ([#119](https://github.com/santosr2/TerraTidy/pull/119)) by [@santosr2](https://github.com/santosr2)
- Float Go version to 1.26.x with check-latest ([#170](https://github.com/santosr2/TerraTidy/pull/170)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Push dependabot lockfile update to PR branch ([#181](https://github.com/santosr2/TerraTidy/pull/181)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Check out branch head for dependabot lockfile commit ([#182](https://github.com/santosr2/TerraTidy/pull/182)) by [@santosr2](https://github.com/santosr2)
- **fuzz,benchmark**: Scope pull-requests: write to job level ([#209](https://github.com/santosr2/TerraTidy/pull/209)) by [@santosr2](https://github.com/santosr2)
- **pip**: Hash-pin installs and add dependabot ecosystem ([#216](https://github.com/santosr2/TerraTidy/pull/216)) by [@santosr2](https://github.com/santosr2)
- **security**: Run SAST on all pushes and PRs ([#220](https://github.com/santosr2/TerraTidy/pull/220)) by [@santosr2](https://github.com/santosr2)
- Harden scorecard concurrency and document test cache choice ([#221](https://github.com/santosr2/TerraTidy/pull/221)) by [@santosr2](https://github.com/santosr2)
- **release**: Use gh auth setup-git for changelog and tag pushes ([#242](https://github.com/santosr2/TerraTidy/pull/242)) by [@santosr2](https://github.com/santosr2)
- Publish the VS Code extension from the release workflow ([84369a0](https://github.com/santosr2/TerraTidy/commit/84369a038cd05f304ca246a45531087a791a2480)) by [@santosr2](https://github.com/santosr2)
- **action**: Float Go version to 1.26.x ([c01d91a](https://github.com/santosr2/TerraTidy/commit/c01d91a7071c02cee5817ef4406ee525d2c39617)) by [@santosr2](https://github.com/santosr2)

### Changed

- **sdk**: Redesign Context with embedded context.Context ([#86](https://github.com/santosr2/TerraTidy/pull/86)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Replace hcl.Range with sdk.Location ([#87](https://github.com/santosr2/TerraTidy/pull/87)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Replace FixFunc closure with FixResult struct ([#88](https://github.com/santosr2/TerraTidy/pull/88)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Add Severity.Level() method ([#89](https://github.com/santosr2/TerraTidy/pull/89)) by [@santosr2](https://github.com/santosr2)
- **config**: Add typed engine configs with ConfigFromEngine converters ([#91](https://github.com/santosr2/TerraTidy/pull/91)) by [@santosr2](https://github.com/santosr2)
- **config**: Remove custom_rules in favor of overrides.rules ([#92](https://github.com/santosr2/TerraTidy/pull/92)) by [@santosr2](https://github.com/santosr2)
- Improve code quality across core components ([#107](https://github.com/santosr2/TerraTidy/pull/107)) by [@santosr2](https://github.com/santosr2)
- **output**: Ensure deterministic output and remove dead code ([#108](https://github.com/santosr2/TerraTidy/pull/108)) by [@santosr2](https://github.com/santosr2)
- Code quality improvements and dead code removal ([#109](https://github.com/santosr2/TerraTidy/pull/109)) by [@santosr2](https://github.com/santosr2)
- Test quality improvements and LSP fix ([#111](https://github.com/santosr2/TerraTidy/pull/111)) by [@santosr2](https://github.com/santosr2)
- **cli**: Improve CLI UX and flag naming ([#112](https://github.com/santosr2/TerraTidy/pull/112)) by [@santosr2](https://github.com/santosr2)
- **style**: Own Fixable signal in engine, dispatch fixes lazily ([#140](https://github.com/santosr2/TerraTidy/pull/140)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Replace Finding.Fix *FixResult with Fixable bool ([#141](https://github.com/santosr2/TerraTidy/pull/141)) by [@santosr2](https://github.com/santosr2)
- **style**: Remove 16 no-op Fix methods from non-fixer rules ([#142](https://github.com/santosr2/TerraTidy/pull/142)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Return *FixResult with byte-range TextEdits from Fixer.Fix ([684493a](https://github.com/santosr2/TerraTidy/commit/684493ab24aa89e8b6b2c0eaa5a16bd38c9b14ca)) by [@santosr2](https://github.com/santosr2)
- **lsp**: Aggregate format-fallback CodeActions ([d31dae6](https://github.com/santosr2/TerraTidy/commit/d31dae61753ec849c74fc125bbe36f74d260a19e)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Return byte-range TextEdits from Fixer.Fix ([#166](https://github.com/santosr2/TerraTidy/pull/166)) by [@santosr2](https://github.com/santosr2)
- **cst**: Preserve nested-block indentation, drop dirty-walk ([#184](https://github.com/santosr2/TerraTidy/pull/184)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite depends-on-order Fix on top of CST ([#190](https://github.com/santosr2/TerraTidy/pull/190)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite ordering Fixes on top of CST ([#191](https://github.com/santosr2/TerraTidy/pull/191)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite provider-block-order Fix on top of CST ([#192](https://github.com/santosr2/TerraTidy/pull/192)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite attribute-group-spacing Fix on top of CST ([#193](https://github.com/santosr2/TerraTidy/pull/193)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite meta-arguments-order Fix on top of CST ([#194](https://github.com/santosr2/TerraTidy/pull/194)) by [@santosr2](https://github.com/santosr2)
- **style**: Rewrite lifecycle-attribute-order Fix on top of CST ([#195](https://github.com/santosr2/TerraTidy/pull/195)) by [@santosr2](https://github.com/santosr2)
- **style**: Drop unused pre-CST helpers and stale references ([#196](https://github.com/santosr2/TerraTidy/pull/196)) by [@santosr2](https://github.com/santosr2)
- **lsp,vscode**: Remove dead on-save settings ([#206](https://github.com/santosr2/TerraTidy/pull/206)) by [@santosr2](https://github.com/santosr2)
- **style**: Make snake_case explicit in naming case switch ([#208](https://github.com/santosr2/TerraTidy/pull/208)) by [@santosr2](https://github.com/santosr2)
- Drop error returns from functions that never fail ([#222](https://github.com/santosr2/TerraTidy/pull/222)) by [@santosr2](https://github.com/santosr2)
- Drop redundant wrappers over sdk file helpers ([#225](https://github.com/santosr2/TerraTidy/pull/225)) by [@santosr2](https://github.com/santosr2)
- **style**: Unexport internal rule helpers ([#226](https://github.com/santosr2/TerraTidy/pull/226)) by [@santosr2](https://github.com/santosr2)
- **style**: Rename rule files to describe their contents ([#227](https://github.com/santosr2/TerraTidy/pull/227)) by [@santosr2](https://github.com/santosr2)

### Dependencies

- **deps**: Bump github.com/zclconf/go-cty from 1.17.0 to 1.18.0 ([#114](https://github.com/santosr2/TerraTidy/pull/114)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump docker/login-action from 4.0.0 to 4.1.0 ([#117](https://github.com/santosr2/TerraTidy/pull/117)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/github-script from 7.0.1 to 8.0.0 ([#116](https://github.com/santosr2/TerraTidy/pull/116)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump esbuild from 0.27.7 to 0.28.0 in /vscode ([#115](https://github.com/santosr2/TerraTidy/pull/115)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump alpine from `2510918` to `5b10f43` ([#122](https://github.com/santosr2/TerraTidy/pull/122)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/upload-artifact from 4.6.2 to 7.0.1 ([#123](https://github.com/santosr2/TerraTidy/pull/123)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.35.0 to 0.36.0 ([#124](https://github.com/santosr2/TerraTidy/pull/124)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/upload-pages-artifact from 4.0.0 to 5.0.0 ([#125](https://github.com/santosr2/TerraTidy/pull/125)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.15.1 to 1.15.2 ([#126](https://github.com/santosr2/TerraTidy/pull/126)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/github-script from 7.0.1 to 9.0.0 ([#127](https://github.com/santosr2/TerraTidy/pull/127)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/cache from 5.0.4 to 5.0.5 ([#128](https://github.com/santosr2/TerraTidy/pull/128)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/setup-python from 5.6.0 to 6.2.0 ([#129](https://github.com/santosr2/TerraTidy/pull/129)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/download-artifact from 4.3.0 to 8.0.1 ([#133](https://github.com/santosr2/TerraTidy/pull/133)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/zclconf/go-cty from 1.18.0 to 1.18.1 ([#132](https://github.com/santosr2/TerraTidy/pull/132)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump goreleaser/goreleaser-action from 7.0.0 to 7.1.0 ([#136](https://github.com/santosr2/TerraTidy/pull/136)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.35.1 to 4.35.2 ([#135](https://github.com/santosr2/TerraTidy/pull/135)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump zizmorcore/zizmor-action from 0.5.2 to 0.5.3 ([#134](https://github.com/santosr2/TerraTidy/pull/134)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump aquasecurity/trivy-action from 0.35.0 to 0.36.0 ([#139](https://github.com/santosr2/TerraTidy/pull/139)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump orhun/git-cliff-action from 4.7.1 to 4.8.0 ([#138](https://github.com/santosr2/TerraTidy/pull/138)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump goreleaser/goreleaser-action from 7.1.0 to 7.2.1 ([#137](https://github.com/santosr2/TerraTidy/pull/137)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.35.2 to 4.35.3 ([#150](https://github.com/santosr2/TerraTidy/pull/150)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/fsnotify/fsnotify from 1.9.0 to 1.10.1 ([#148](https://github.com/santosr2/TerraTidy/pull/148)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.15.2 to 1.16.1 ([#149](https://github.com/santosr2/TerraTidy/pull/149)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump sigstore/cosign-installer from 4.1.1 to 4.1.2 ([#157](https://github.com/santosr2/TerraTidy/pull/157)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.35.3 to 4.35.4 ([#156](https://github.com/santosr2/TerraTidy/pull/156)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.36.0 to 0.37.0 ([#155](https://github.com/santosr2/TerraTidy/pull/155)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/dependency-review-action from 4.9.0 to 5.0.0 ([#154](https://github.com/santosr2/TerraTidy/pull/154)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.35.4 to 4.35.5 ([#165](https://github.com/santosr2/TerraTidy/pull/165)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump zizmorcore/zizmor-action from 0.5.3 to 0.5.6 ([#164](https://github.com/santosr2/TerraTidy/pull/164)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump codecov/codecov-action from 6.0.0 to 6.0.1 ([#163](https://github.com/santosr2/TerraTidy/pull/163)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump goreleaser/goreleaser-action from 7.2.1 to 7.2.2 ([#162](https://github.com/santosr2/TerraTidy/pull/162)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.16.1 to 1.16.2 ([#161](https://github.com/santosr2/TerraTidy/pull/161)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump docker/login-action from 4.1.0 to 4.2.0 ([#168](https://github.com/santosr2/TerraTidy/pull/168)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golangci/golangci-lint-action from 9.2.0 to 9.2.1 ([#169](https://github.com/santosr2/TerraTidy/pull/169)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.35.5 to 4.36.0 ([#167](https://github.com/santosr2/TerraTidy/pull/167)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.16.2 to 1.17.0 ([#171](https://github.com/santosr2/TerraTidy/pull/171)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump imjasonh/setup-crane from 0.5 to 0.6 ([#172](https://github.com/santosr2/TerraTidy/pull/172)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump gitleaks/gitleaks-action from 2.3.9 to 3.0.0 ([#173](https://github.com/santosr2/TerraTidy/pull/173)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump alpine from 3.23 to 3.24 ([#174](https://github.com/santosr2/TerraTidy/pull/174)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/checkout from 6.0.2 to 6.0.3 ([#175](https://github.com/santosr2/TerraTidy/pull/175)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.37.0 to 0.38.0 ([#176](https://github.com/santosr2/TerraTidy/pull/176)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.17.0 to 1.17.1 ([#177](https://github.com/santosr2/TerraTidy/pull/177)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action from 4.36.0 to 4.36.2 ([#179](https://github.com/santosr2/TerraTidy/pull/179)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump codecov/codecov-action from 6.0.1 to 7.0.0 ([#180](https://github.com/santosr2/TerraTidy/pull/180)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump vscode-languageclient from 9.0.1 to 10.0.0 in /vscode ([#178](https://github.com/santosr2/TerraTidy/pull/178)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump alpine from 3.24 to 3.24.1 ([#187](https://github.com/santosr2/TerraTidy/pull/187)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @vscode/test-electron from 2.5.2 to 3.0.0 in /vscode ([#188](https://github.com/santosr2/TerraTidy/pull/188)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump zizmorcore/zizmor-action from 0.5.6 to 0.5.7 ([#203](https://github.com/santosr2/TerraTidy/pull/203)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/checkout from 6.0.3 to 7.0.0 ([#205](https://github.com/santosr2/TerraTidy/pull/205)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @types/node from 25.9.4 to 26.0.0 in /vscode ([#204](https://github.com/santosr2/TerraTidy/pull/204)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.17.1 to 1.18.1 ([#210](https://github.com/santosr2/TerraTidy/pull/210)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump goreleaser/goreleaser-action from 7.2.2 to 7.2.3 ([#211](https://github.com/santosr2/TerraTidy/pull/211)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump imjasonh/setup-crane from 0.6 to 0.7 ([#212](https://github.com/santosr2/TerraTidy/pull/212)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/setup-python from 6.2.0 to 6.3.0 ([#213](https://github.com/santosr2/TerraTidy/pull/213)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/cache from 5.0.5 to 6.1.0 ([#215](https://github.com/santosr2/TerraTidy/pull/215)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/setup-go from 6.4.0 to 6.5.0 ([#214](https://github.com/santosr2/TerraTidy/pull/214)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golangci/golangci-lint-action from 9.2.1 to 9.3.0 ([#219](https://github.com/santosr2/TerraTidy/pull/219)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/attest-build-provenance from 4.1.0 to 4.1.1 ([#218](https://github.com/santosr2/TerraTidy/pull/218)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump pymdown-extensions from 10.21.2 to 11.0 in /docs/site ([#217](https://github.com/santosr2/TerraTidy/pull/217)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump docker/login-action from 4.2.0 to 4.4.0 ([#233](https://github.com/santosr2/TerraTidy/pull/233)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action/upload-sarif from 4.36.2 to 4.36.3 ([#237](https://github.com/santosr2/TerraTidy/pull/237)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump pymdown-extensions from 11.0 to 11.0.1 in /docs/site ([#236](https://github.com/santosr2/TerraTidy/pull/236)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.38.0 to 0.39.0 ([#235](https://github.com/santosr2/TerraTidy/pull/235)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.18.1 to 1.18.2 ([#234](https://github.com/santosr2/TerraTidy/pull/234)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/zclconf/go-cty from 1.18.1 to 1.19.0 ([#243](https://github.com/santosr2/TerraTidy/pull/243)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang/govulncheck-action from 1.0.4 to 1.1.0 ([#245](https://github.com/santosr2/TerraTidy/pull/245)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.39.0 to 0.40.0 ([#244](https://github.com/santosr2/TerraTidy/pull/244)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github/codeql-action/upload-sarif from 4.36.3 to 4.37.0 ([#247](https://github.com/santosr2/TerraTidy/pull/247)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump typescript from 6.0.3 to 7.0.2 in /vscode ([#246](https://github.com/santosr2/TerraTidy/pull/246)) by [@dependabot[bot]](https://github.com/dependabot[bot])

### Documentation

- Update changelog for v0.2.0-alpha.4 ([d267e60](https://github.com/santosr2/TerraTidy/commit/d267e60b2d44c4f94e4718be8a970ec0f16e2778)) by [@github-actions[bot]](https://github.com/github-actions[bot])
- Comprehensive documentation and examples review ([#104](https://github.com/santosr2/TerraTidy/pull/104)) by [@santosr2](https://github.com/santosr2)
- Add v0.1.0 deprecation notice and discussion templates ([#106](https://github.com/santosr2/TerraTidy/pull/106)) by [@santosr2](https://github.com/santosr2)
- Add performance guide, update architecture, and refresh baseline ([9094af6](https://github.com/santosr2/TerraTidy/commit/9094af6d110de7d6ea16cb944f7bff5200c1dfba)) by [@santosr2](https://github.com/santosr2)
- Update configuration and upgrade guides ([6a08b95](https://github.com/santosr2/TerraTidy/commit/6a08b955137320afe8b127630b73807dde53f380)) by [@santosr2](https://github.com/santosr2)
- **rules**: Align style-rules page with #151 behavior ([#152](https://github.com/santosr2/TerraTidy/pull/152)) by [@santosr2](https://github.com/santosr2)
- Notes on SDK migration, user-guide sync, and fmt/style test polish ([#160](https://github.com/santosr2/TerraTidy/pull/160)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Document byte-range TextEdit Fix contract and migration ([f97ccda](https://github.com/santosr2/TerraTidy/commit/f97ccdaf298ae12c12a740047e057d751b3020ad)) by [@santosr2](https://github.com/santosr2)
- **upgrade**: Annotate the v0.2.0-alpha.5 historical Fix signature ([eb18a6d](https://github.com/santosr2/TerraTidy/commit/eb18a6dfee5bc982adc3ba9562320433e605bc23)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Clarify TextEdit godoc on End invariant and Replacement JSON encoding ([a08c79c](https://github.com/santosr2/TerraTidy/commit/a08c79cfa40022cef702db55096afb8f2a2896c6)) by [@santosr2](https://github.com/santosr2)
- **architecture**: Inline apply-order detail in engine pipeline paragraph ([1ccb4cb](https://github.com/santosr2/TerraTidy/commit/1ccb4cbe91a3c6cf0ddb6ca31f9db93cf74c54b8)) by [@santosr2](https://github.com/santosr2)
- **cst**: Document CST package and ship terragrunt-include-first example ([#198](https://github.com/santosr2/TerraTidy/pull/198)) by [@santosr2](https://github.com/santosr2)
- **style**: Align severity table + polish stale comments ([#200](https://github.com/santosr2/TerraTidy/pull/200)) by [@santosr2](https://github.com/santosr2)
- **ci**: Correct workflow trigger table + add vscode.yml row ([#201](https://github.com/santosr2/TerraTidy/pull/201)) by [@santosr2](https://github.com/santosr2)
- Refresh architecture, configuration, and commands pages ([#202](https://github.com/santosr2/TerraTidy/pull/202)) by [@santosr2](https://github.com/santosr2)
- **examples**: Document plugin integrity + refresh integration workflow bumps ([#207](https://github.com/santosr2/TerraTidy/pull/207)) by [@santosr2](https://github.com/santosr2)
- **plugins**: Link bash rule verification TODO to #228 ([#230](https://github.com/santosr2/TerraTidy/pull/230)) by [@santosr2](https://github.com/santosr2)
- **examples**: Add complete config with all features enabled ([#241](https://github.com/santosr2/TerraTidy/pull/241)) by [@santosr2](https://github.com/santosr2)
- Correct references to the unreleased v0.2.0-alpha.5 version ([fe14098](https://github.com/santosr2/TerraTidy/commit/fe14098fa69e63f410c1fbdc79183f35a7c7472d)) by [@santosr2](https://github.com/santosr2)
- Drop @latest deprecation notices and sync docs for v0.2.0 ([4cb9831](https://github.com/santosr2/TerraTidy/commit/4cb9831d1b578548746c79419dd9747bfcd0891d)) by [@santosr2](https://github.com/santosr2)
- Adopt Contributor Covenant 3.0 and fix the contributor guide ([775be0f](https://github.com/santosr2/TerraTidy/commit/775be0f965a0a699cb69731f745d65f9bd927086)) by [@santosr2](https://github.com/santosr2)
- Document the stable-only VS Code Marketplace policy ([54132c6](https://github.com/santosr2/TerraTidy/commit/54132c61c634900bb71ac6afd7b8019a1c5d67fc)) by [@santosr2](https://github.com/santosr2)
- Fix custom-rule testing steps and stale Go/OPA versions ([1c1ff8a](https://github.com/santosr2/TerraTidy/commit/1c1ff8aa5ca400c679215ef23825ff9fd5273f99)) by [@santosr2](https://github.com/santosr2)
- **readme**: Correct default-mode quick-start output ([22b3ad8](https://github.com/santosr2/TerraTidy/commit/22b3ad8ccb2499bedd5d0cef72955fddaddae141)) by [@santosr2](https://github.com/santosr2)
- Fix clone directory case in contributing guide ([c24a535](https://github.com/santosr2/TerraTidy/commit/c24a535d738f001491aa3a17dae07ce7d85d3f15)) by [@santosr2](https://github.com/santosr2)
- **output-formats**: Regenerate examples from real CLI output ([4f2883a](https://github.com/santosr2/TerraTidy/commit/4f2883afd81a84d08ad140bb255803846f01b1f3)) by [@santosr2](https://github.com/santosr2)
- Correct release-facing inaccuracies ([c50b1b7](https://github.com/santosr2/TerraTidy/commit/c50b1b7cb010c76e1c26bd8669a2fcb49a604b48)) by [@santosr2](https://github.com/santosr2)
- Correct data_files format and engine package name ([1c46e5f](https://github.com/santosr2/TerraTidy/commit/1c46e5fc150a8ccf8d95877c77ad4f43fdd1b9b9)) by [@santosr2](https://github.com/santosr2)

### Fixed

- Resolve 7 critical bugs across engines and CLI ([#90](https://github.com/santosr2/TerraTidy/pull/90)) by [@santosr2](https://github.com/santosr2)
- **ci**: Update existing benchmark comment instead of creating new ([#97](https://github.com/santosr2/TerraTidy/pull/97)) by [@santosr2](https://github.com/santosr2)
- **output**: Always include line:col in text formatter ([e80468c](https://github.com/santosr2/TerraTidy/commit/e80468c61949612602e679dc9087ab19ff3627fb)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve comments during attribute reordering ([#130](https://github.com/santosr2/TerraTidy/pull/130)) by [@santosr2](https://github.com/santosr2)
- **cli**: Make fmt and style diff flags work consistently ([#131](https://github.com/santosr2/TerraTidy/pull/131)) by [@santosr2](https://github.com/santosr2)
- **style**: Improve rule accuracy, preserve comments and nested blocks ([#151](https://github.com/santosr2/TerraTidy/pull/151)) by [@santosr2](https://github.com/santosr2)
- **cli**: Print error and exit 3 on missing target file ([#159](https://github.com/santosr2/TerraTidy/pull/159)) by [@santosr2](https://github.com/santosr2)
- **lsp**: Use UTF-16 code units for Position.character ([76d518d](https://github.com/santosr2/TerraTidy/commit/76d518d8acd20c8fcacb70ad00a4c3f94eb3bd2b)) by [@santosr2](https://github.com/santosr2)
- **style**: Make MetaArgumentsOrderRule.Fix a no-op on canonical input ([d14e9d5](https://github.com/santosr2/TerraTidy/commit/d14e9d55612b7143b27cf3d8d4bd2727310ef7fb)) by [@santosr2](https://github.com/santosr2)
- **style**: Relocate tags above lifecycle when authored below ([#185](https://github.com/santosr2/TerraTidy/pull/185)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve floating section comments on terraform-block reorder ([#186](https://github.com/santosr2/TerraTidy/pull/186)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve leading comment on lifecycle reorder ([#189](https://github.com/santosr2/TerraTidy/pull/189)) by [@santosr2](https://github.com/santosr2)
- **docs**: Close unclosed code fence + wire CST into project setup ([#199](https://github.com/santosr2/TerraTidy/pull/199)) by [@santosr2](https://github.com/santosr2)
- **config**: Accumulate plugins.tags across imports ([#224](https://github.com/santosr2/TerraTidy/pull/224)) by [@santosr2](https://github.com/santosr2)
- **config**: Fail on unset ${VAR:?} required environment variables ([e8ae637](https://github.com/santosr2/TerraTidy/commit/e8ae6371426bc36b3acf2a26ecd7f71dcdffbb10)) by [@santosr2](https://github.com/santosr2)
- **action**: Fail on tool errors and honor parallel:false ([04ef99a](https://github.com/santosr2/TerraTidy/commit/04ef99a011a7a01db8f6580368b191d4e4dd5175)) by [@santosr2](https://github.com/santosr2)
- **release**: Correct VS Code Marketplace link in release notes ([c1b82ee](https://github.com/santosr2/TerraTidy/commit/c1b82ee831f884767a93318d3daf43e06f51ef39)) by [@santosr2](https://github.com/santosr2)
- **ci**: Smoke-test the published Homebrew formula, not the tag's ([2b53ed8](https://github.com/santosr2/TerraTidy/commit/2b53ed8c4535ce5bf23a2260f7fd33847b550481)) by [@santosr2](https://github.com/santosr2)
- **output**: Use relative display path for JUnit testsuite name ([cf11cd4](https://github.com/santosr2/TerraTidy/commit/cf11cd4d9a0a8c6877abb51116d18c743b20e877)) by [@santosr2](https://github.com/santosr2)
- **release**: Keep output-formats version strings tracking current_version ([264f89e](https://github.com/santosr2/TerraTidy/commit/264f89e01538ce18424c7a0cc8d5b6038d911fca)) by [@santosr2](https://github.com/santosr2)
- **action**: Upload SARIF results even when findings fail the run ([28ea4f4](https://github.com/santosr2/TerraTidy/commit/28ea4f4b9890053f197cef9671dd79fe0272ae71)) by [@santosr2](https://github.com/santosr2)
- **plugins**: Import pkg/plugins in the init scaffold and guide ([52df705](https://github.com/santosr2/TerraTidy/commit/52df705099a9d86cfd1eba08ea6ae97543e8ac44)) by [@santosr2](https://github.com/santosr2)

### Other

- Benchmark distribution, example testing, and cleanup ([#96](https://github.com/santosr2/TerraTidy/pull/96)) by [@santosr2](https://github.com/santosr2)
- Improve error messages and fix CI deprecations ([#113](https://github.com/santosr2/TerraTidy/pull/113)) by [@santosr2](https://github.com/santosr2)
- Add pre-commit linters for workflows, Dockerfile, and shell scripts ([2403201](https://github.com/santosr2/TerraTidy/commit/24032013d035e97802d422e041f1170e17b84ada)) by [@santosr2](https://github.com/santosr2)
- Update gitignore, CLAUDE.md, and CONTRIBUTING.md ([a43bd3f](https://github.com/santosr2/TerraTidy/commit/a43bd3fb81ed7bdefcf92007f3b8a90ffbb64ec7)) by [@santosr2](https://github.com/santosr2)
- Comprehensive quality improvements ([#118](https://github.com/santosr2/TerraTidy/pull/118)) by [@santosr2](https://github.com/santosr2)
- Bump pinned Go toolchain from 1.26.1 to 1.26.3 ([#147](https://github.com/santosr2/TerraTidy/pull/147)) by [@santosr2](https://github.com/santosr2)
- **examples**: Bump go-rule indirect deps and document go mod tidy ([923c759](https://github.com/santosr2/TerraTidy/commit/923c75925fa257d54a2f83ee687a63a0e5ff25ef)) by [@santosr2](https://github.com/santosr2)
- **mise**: Add docs:serve task for local doc preview ([1d75433](https://github.com/santosr2/TerraTidy/commit/1d754332d449c0fa84c59881cfc55ee047a13a98)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Pin bun to 1.2.x and refresh lockfile ([b3ed261](https://github.com/santosr2/TerraTidy/commit/b3ed261ec83e1eb4de4ccb215a26809f2ffbdf54)) by [@santosr2](https://github.com/santosr2)
- Prepare 0.2.0 release ([#248](https://github.com/santosr2/TerraTidy/pull/248)) by [@santosr2](https://github.com/santosr2)

### VS Code Extension

- Switch to pattern-based activation for extension compatibility ([#110](https://github.com/santosr2/TerraTidy/pull/110)) by [@santosr2](https://github.com/santosr2)
- Add error handling and comprehensive test coverage ([0da7bbd](https://github.com/santosr2/TerraTidy/commit/0da7bbd992520b8d00fa67c35711a6dc810644aa)) by [@santosr2](https://github.com/santosr2)
- **release**: Decouple versioning and enhance action outputs ([7e4142a](https://github.com/santosr2/TerraTidy/commit/7e4142a2d2513a3a3d018fbdc18144aa4090c0d3)) by [@santosr2](https://github.com/santosr2)
- Override undici to ^7.28.0 to clear transitive CVEs ([#231](https://github.com/santosr2/TerraTidy/pull/231)) by [@santosr2](https://github.com/santosr2)
- Bump esbuild and override 7 transitive deps to clear CVEs ([#232](https://github.com/santosr2/TerraTidy/pull/232)) by [@santosr2](https://github.com/santosr2)
- Override diff and serialize-javascript to clear CVEs ([#238](https://github.com/santosr2/TerraTidy/pull/238)) by [@santosr2](https://github.com/santosr2)
- Regenerate lockfile to clear remaining transitive CVEs ([#239](https://github.com/santosr2/TerraTidy/pull/239)) by [@santosr2](https://github.com/santosr2)
- Bump extension to 0.2.1 and pin moduleResolution for TypeScript 7 ([dedf26c](https://github.com/santosr2/TerraTidy/commit/dedf26ce157790fe9b8640a24bdfe1270c1e9777)) by [@santosr2](https://github.com/santosr2)

### Build

- Bump Go toolchain to 1.26.4 ([#223](https://github.com/santosr2/TerraTidy/pull/223)) by [@santosr2](https://github.com/santosr2)
- Replace bump:release with bump:stage and bump:stable ([ee9b76f](https://github.com/santosr2/TerraTidy/commit/ee9b76f2d02bc8c9d184e765b86dbbb9879618e0)) by [@santosr2](https://github.com/santosr2)
- **release**: Publish multi-arch amd64/arm64 docker images ([bc1606d](https://github.com/santosr2/TerraTidy/commit/bc1606d782ed1cdb7f6e0fd5c3773df62370f2a3)) by [@santosr2](https://github.com/santosr2)
- **release**: Track discussion template version placeholder ([d918e91](https://github.com/santosr2/TerraTidy/commit/d918e91f0a381153382c4e64c4b162f83f0f9927)) by [@santosr2](https://github.com/santosr2)
- **release**: Drop stale CHANGELOG.md from release archives ([4487b6c](https://github.com/santosr2/TerraTidy/commit/4487b6cd1fea500a317a63ab41cbb5a21d4e6856)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha.4] - 2026-04-04

### Added

- **plugins**: Add YAML and Bash rule loaders with examples ([#4](https://github.com/santosr2/TerraTidy/pull/4)) by [@santosr2](https://github.com/santosr2)

### CI/CD

- **release**: Mark pre-releases as latest on GitHub ([8f2a130](https://github.com/santosr2/TerraTidy/commit/8f2a13026e21d4a64b4ecdebe3613ad8a6fe134a)) by [@santosr2](https://github.com/santosr2)
- Add dependabot and development utility scripts ([#5](https://github.com/santosr2/TerraTidy/pull/5)) by [@santosr2](https://github.com/santosr2)
- Add permissions, timeouts, concurrency, and path filters to workflows ([#35](https://github.com/santosr2/TerraTidy/pull/35)) by [@santosr2](https://github.com/santosr2)
- Fix injection risks, pin actions to SHAs, and harden Dockerfile ([#36](https://github.com/santosr2/TerraTidy/pull/36)) by [@santosr2](https://github.com/santosr2)
- Add security scanning, quality checks, and pin mise tool versions ([#38](https://github.com/santosr2/TerraTidy/pull/38)) by [@santosr2](https://github.com/santosr2)
- Add SBOM, cosign signing, attestations, and OpenSSF Scorecard ([#39](https://github.com/santosr2/TerraTidy/pull/39)) by [@santosr2](https://github.com/santosr2)
- Improve release pipeline with shared version parsing, crane, and smoke tests ([#40](https://github.com/santosr2/TerraTidy/pull/40)) by [@santosr2](https://github.com/santosr2)
- **quality**: Cache pre-commit hooks and skip branch guard in CI ([#45](https://github.com/santosr2/TerraTidy/pull/45)) by [@santosr2](https://github.com/santosr2)
- Add PR title checker and standardize commit conventions ([#51](https://github.com/santosr2/TerraTidy/pull/51)) by [@santosr2](https://github.com/santosr2)

### Changed

- **style**: Remove documentation rules duplicated by lint engine ([#3](https://github.com/santosr2/TerraTidy/pull/3)) by [@santosr2](https://github.com/santosr2)
- Consolidate duplicated utilities and standardize interface{} to any ([#47](https://github.com/santosr2/TerraTidy/pull/47)) by [@santosr2](https://github.com/santosr2)
- **sdk**: Split Rule/Fixer interfaces, unify Engine, remove Context.Logger ([#58](https://github.com/santosr2/TerraTidy/pull/58)) by [@santosr2](https://github.com/santosr2)
- **build**: Consolidate Makefile into mise tasks ([#63](https://github.com/santosr2/TerraTidy/pull/63)) by [@santosr2](https://github.com/santosr2)

### Dependencies

- **deps**: Bump actions/cache from 4 to 5 ([#6](https://github.com/santosr2/TerraTidy/pull/6)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/upload-pages-artifact from 3 to 4 ([#7](https://github.com/santosr2/TerraTidy/pull/7)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/setup-python from 5 to 6 ([#8](https://github.com/santosr2/TerraTidy/pull/8)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump goreleaser/goreleaser-action from 6 to 7 ([#9](https://github.com/santosr2/TerraTidy/pull/9)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump docker/login-action from 3 to 4 ([#11](https://github.com/santosr2/TerraTidy/pull/11)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.12.0 to 1.15.0 ([#10](https://github.com/santosr2/TerraTidy/pull/10)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @types/node from 20.19.37 to 25.5.0 in /vscode ([#12](https://github.com/santosr2/TerraTidy/pull/12)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump golang.org/x/text from 0.34.0 to 0.35.0 ([#13](https://github.com/santosr2/TerraTidy/pull/13)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump typescript from 5.9.3 to 6.0.2 in /vscode ([#14](https://github.com/santosr2/TerraTidy/pull/14)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @biomejs/biome from 1.9.4 to 2.4.9 in /vscode ([#16](https://github.com/santosr2/TerraTidy/pull/16)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @vscode/vsce from 2.32.0 to 3.7.1 in /vscode ([#15](https://github.com/santosr2/TerraTidy/pull/15)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/checkout from 4 to 6 ([#30](https://github.com/santosr2/TerraTidy/pull/30)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump github.com/open-policy-agent/opa from 1.15.0 to 1.15.1 ([#32](https://github.com/santosr2/TerraTidy/pull/32)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump alpine from 3.21 to 3.23 ([#37](https://github.com/santosr2/TerraTidy/pull/37)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/cache from 4.3.0 to 5.0.4 ([#52](https://github.com/santosr2/TerraTidy/pull/52)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump actions/upload-artifact from 4.6.2 to 7.0.0 ([#73](https://github.com/santosr2/TerraTidy/pull/73)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump anchore/sbom-action from 0.20.0 to 0.24.0 ([#75](https://github.com/santosr2/TerraTidy/pull/75)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps**: Bump oven-sh/setup-bun from 2.0.2 to 2.2.0 ([#77](https://github.com/santosr2/TerraTidy/pull/77)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump esbuild from 0.25.12 to 0.27.4 in /vscode ([#76](https://github.com/santosr2/TerraTidy/pull/76)) by [@dependabot[bot]](https://github.com/dependabot[bot])
- **deps-dev**: Bump @types/node from 22.19.15 to 25.5.0 in /vscode ([#74](https://github.com/santosr2/TerraTidy/pull/74)) by [@dependabot[bot]](https://github.com/dependabot[bot])

### Documentation

- Update changelog for v0.2.0-alpha.3 ([f188998](https://github.com/santosr2/TerraTidy/commit/f188998f7c6d633d1485cba6c8340d72362d31eb)) by [@github-actions[bot]](https://github.com/github-actions[bot])
- Align documentation with actual implementation ([#18](https://github.com/santosr2/TerraTidy/pull/18)) by [@santosr2](https://github.com/santosr2)
- Fix version refs, dead links, and missing output formats ([#21](https://github.com/santosr2/TerraTidy/pull/21)) by [@santosr2](https://github.com/santosr2)
- Document style rules, commands, and output formats ([#24](https://github.com/santosr2/TerraTidy/pull/24)) by [@santosr2](https://github.com/santosr2)
- Document config features, plugin system, and custom rules ([#25](https://github.com/santosr2/TerraTidy/pull/25)) by [@santosr2](https://github.com/santosr2)
- Document policy rules, LSP capabilities, and test-rule format ([#26](https://github.com/santosr2/TerraTidy/pull/26)) by [@santosr2](https://github.com/santosr2)
- Create 9 new documentation pages and enhance SDK godoc ([#27](https://github.com/santosr2/TerraTidy/pull/27)) by [@santosr2](https://github.com/santosr2)
- Polish navigation, fix pre-commit hooks, and add clarifications ([#28](https://github.com/santosr2/TerraTidy/pull/28)) by [@santosr2](https://github.com/santosr2)
- Fix 16 factual inaccuracies across documentation site ([#29](https://github.com/santosr2/TerraTidy/pull/29)) by [@santosr2](https://github.com/santosr2)
- **readme**: Fix output examples, flag docs, and bumpversion coverage ([#54](https://github.com/santosr2/TerraTidy/pull/54)) by [@santosr2](https://github.com/santosr2)
- Fix README inaccuracies, broken Homebrew install, and plugin template bug ([#55](https://github.com/santosr2/TerraTidy/pull/55)) by [@santosr2](https://github.com/santosr2)
- Sync documentation with recent code changes ([#57](https://github.com/santosr2/TerraTidy/pull/57)) by [@santosr2](https://github.com/santosr2)
- **lint**: Clarify TFLint subprocess integration and add license note ([#67](https://github.com/santosr2/TerraTidy/pull/67)) by [@santosr2](https://github.com/santosr2)

### Fixed

- Use dynamic version in LSP and fix plugin scaffold ([#19](https://github.com/santosr2/TerraTidy/pull/19)) by [@santosr2](https://github.com/santosr2)
- Remove false claims, fix examples, and implement --diff flag ([#23](https://github.com/santosr2/TerraTidy/pull/23)) by [@santosr2](https://github.com/santosr2)
- Replace os.Exit with error returns and consolidate SDK utilities ([#46](https://github.com/santosr2/TerraTidy/pull/46)) by [@santosr2](https://github.com/santosr2)
- Add LSP write mutex, config cycle detection, and interface cleanup ([#56](https://github.com/santosr2/TerraTidy/pull/56)) by [@santosr2](https://github.com/santosr2)
- Toolchain, LSP features, pre-commit hooks, and VSCode security ([#60](https://github.com/santosr2/TerraTidy/pull/60)) by [@santosr2](https://github.com/santosr2)
- Docker, GitHub Actions, VSCode, and LSP audit fixes ([#61](https://github.com/santosr2/TerraTidy/pull/61)) by [@santosr2](https://github.com/santosr2)
- **lsp**: Prevent path traversal attacks in URI handling ([#68](https://github.com/santosr2/TerraTidy/pull/68)) by [@santosr2](https://github.com/santosr2)
- **security**: Harden code execution paths ([#69](https://github.com/santosr2/TerraTidy/pull/69)) by [@santosr2](https://github.com/santosr2)
- **security**: Add input validation and error sanitization ([#71](https://github.com/santosr2/TerraTidy/pull/71)) by [@santosr2](https://github.com/santosr2)
- **release**: Update bump-my-version config ([#83](https://github.com/santosr2/TerraTidy/pull/83)) by [@santosr2](https://github.com/santosr2)
- **release**: Use cosign bundle format for signing ([#85](https://github.com/santosr2/TerraTidy/pull/85)) by [@santosr2](https://github.com/santosr2)

### Other

- **dev**: Upgrade to Go 1.26.1 and clean up dev tooling ([#17](https://github.com/santosr2/TerraTidy/pull/17)) by [@santosr2](https://github.com/santosr2)
- Add Claude Code project instructions ([#20](https://github.com/santosr2/TerraTidy/pull/20)) by [@santosr2](https://github.com/santosr2)
- Apply linter fixes and add plan dirs to gitignore ([#22](https://github.com/santosr2/TerraTidy/pull/22)) by [@santosr2](https://github.com/santosr2)
- Improve community health templates and add status badges docs ([#42](https://github.com/santosr2/TerraTidy/pull/42)) by [@santosr2](https://github.com/santosr2)
- Add CODEOWNERS, extract action script, optimize test matrix ([#43](https://github.com/santosr2/TerraTidy/pull/43)) by [@santosr2](https://github.com/santosr2)
- Fix Scorecard URL casing and add branch protection hook ([#44](https://github.com/santosr2/TerraTidy/pull/44)) by [@santosr2](https://github.com/santosr2)
- **lint**: Add golangci-lint config, fix security permissions and code findings ([#62](https://github.com/santosr2/TerraTidy/pull/62)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Add tests, fix LSP diagnostics, improve config handling ([#66](https://github.com/santosr2/TerraTidy/pull/66)) by [@santosr2](https://github.com/santosr2)
- Harden CI/CD supply chain and add config enhancements ([#72](https://github.com/santosr2/TerraTidy/pull/72)) by [@santosr2](https://github.com/santosr2)

### Performance

- Add benchmarks for lint, policy, and output formatters ([#59](https://github.com/santosr2/TerraTidy/pull/59)) by [@santosr2](https://github.com/santosr2)

### Revert

- Remove make_latest (GitHub limitation for prereleases) ([15c013c](https://github.com/santosr2/TerraTidy/commit/15c013c71269c3e074f83b14cfb80d0573a02a97)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha.3] - 2026-01-19

### Added

- **build**: Add embedded version info for go install support ([bf17e1c](https://github.com/santosr2/TerraTidy/commit/bf17e1c276e3cb07e3df4d8f77209e3abab218bf)) by [@santosr2](https://github.com/santosr2)
- **fmt**: Add --all flag to combine formatting with style fixes ([0393931](https://github.com/santosr2/TerraTidy/commit/03939314aff84fbfdf46a8ac179e11b4d097e7cd)) by [@santosr2](https://github.com/santosr2)
- **style**: Add attribute group spacing rule ([4f96a76](https://github.com/santosr2/TerraTidy/commit/4f96a76d236a058ea07724b2baaba34ccacbbc07)) by [@santosr2](https://github.com/santosr2)
- **style**: Add one-line vs block attribute spacing rule ([3ea46b3](https://github.com/santosr2/TerraTidy/commit/3ea46b34ffb691f3a33e4724c3cff8a143dcf8da)) by [@santosr2](https://github.com/santosr2)
- Add config wiring, configurable naming, new rules and output formats ([91f2ab5](https://github.com/santosr2/TerraTidy/commit/91f2ab5b09bb91fafb0bef8acd77620055c35578)) by [@santosr2](https://github.com/santosr2)
- **style**: Add 13 new rules and comprehensive documentation ([d6380d5](https://github.com/santosr2/TerraTidy/commit/d6380d5011441dc2f659e6306109a737381edf0a)) by [@santosr2](https://github.com/santosr2)

### CI/CD

- Add one-time workflow to fix Docker alias tags ([61c8f81](https://github.com/santosr2/TerraTidy/commit/61c8f811e744b392ace8bb11e6ed335b00df5f99)) by [@santosr2](https://github.com/santosr2)
- Remove one-time Docker fix workflow ([3ab9f8b](https://github.com/santosr2/TerraTidy/commit/3ab9f8b7dd0e29eaa76fb32cf8d1e6cdc858e512)) by [@santosr2](https://github.com/santosr2)
- Add workflow to fix Docker alias tags ([a7faff1](https://github.com/santosr2/TerraTidy/commit/a7faff11eeb60a6657d84b11aafbd9a17374df4d)) by [@santosr2](https://github.com/santosr2)
- Remove one-time Docker fix workflow ([366979c](https://github.com/santosr2/TerraTidy/commit/366979ca70693210230444924a1babe384dd5592)) by [@santosr2](https://github.com/santosr2)
- Add codecov configuration ([41234a3](https://github.com/santosr2/TerraTidy/commit/41234a3454df44181e7c20fbe4e37b5f78beae63)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog for v0.2.0-alpha.2 ([96c06be](https://github.com/santosr2/TerraTidy/commit/96c06be3823d47ea0255eeee22d912939954ec6e)) by [@github-actions[bot]](https://github.com/github-actions[bot])
- Clarify that 'latest' Docker tag includes pre-releases ([e50ac4c](https://github.com/santosr2/TerraTidy/commit/e50ac4c84a473792adc8237b87c189d89b44936d)) by [@santosr2](https://github.com/santosr2)

### Fixed

- **release**: Improve changelog and add Docker alias tag updates ([3426e86](https://github.com/santosr2/TerraTidy/commit/3426e8606a125ff83e2e28afad24fc175988d44f)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve internal blank lines and re-run fmt after style fixes ([9e76798](https://github.com/santosr2/TerraTidy/commit/9e76798ab18ec6c381f602e0fdbedd7cded073e1)) by [@santosr2](https://github.com/santosr2)
- **cli**: Fix color flag and add changelog links ([b11560c](https://github.com/santosr2/TerraTidy/commit/b11560c1bab748e08cd634e98d31543257db4798)) by [@santosr2](https://github.com/santosr2)
- **cli**: Apply global color flag to all commands ([3f50fbb](https://github.com/santosr2/TerraTidy/commit/3f50fbb473810f03efe5e35ba683f6fa31575b41)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve inline comments when reordering HCL attributes ([#2](https://github.com/santosr2/TerraTidy/pull/2)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha.2] - 2026-01-12

### Added

- **release**: Add automated changelog and fix pre-release handling ([6336002](https://github.com/santosr2/TerraTidy/commit/63360025c80b1c0b8324bb03d5680f7eba9ea762)) by [@santosr2](https://github.com/santosr2)
- Add bump-my-version for version management ([cfabfba](https://github.com/santosr2/TerraTidy/commit/cfabfba767be4383e18077220cf57c6258650e7d)) by [@santosr2](https://github.com/santosr2)

### CI/CD

- Add coverpkg flag for accurate cross-package coverage ([ac0c54c](https://github.com/santosr2/TerraTidy/commit/ac0c54cccb18a1c66ccc987f5c720529313a6e63)) by [@santosr2](https://github.com/santosr2)

### Changed

- **style**: Split rules.go into modular files ([ba5aa31](https://github.com/santosr2/TerraTidy/commit/ba5aa3151c6a2fc17c3e9af258b9db6f77eb541e)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog with v0.2.0-alpha release ([194b1b0](https://github.com/santosr2/TerraTidy/commit/194b1b0b1bc15c4092038bb14035b5f74b80855a)) by [@santosr2](https://github.com/santosr2)
- Add community documentation files ([f38fa57](https://github.com/santosr2/TerraTidy/commit/f38fa57a39b084d56b2d666e885648843e7a94b1)) by [@santosr2](https://github.com/santosr2)
- Update CLI output examples to match actual output ([805da58](https://github.com/santosr2/TerraTidy/commit/805da585db2b617b5325023f692129ccb36d9bc7)) by [@santosr2](https://github.com/santosr2)

### Fixed

- **action**: Embed version info in build from source ([559e7cc](https://github.com/santosr2/TerraTidy/commit/559e7cc3fa6abfa3aa03287f55ab62fce609e61a)) by [@santosr2](https://github.com/santosr2)
- **style**: Implement proper attribute ordering with auto-fix ([516893f](https://github.com/santosr2/TerraTidy/commit/516893f1259bc74535ce3646ace831a3fb24dfdb)) by [@santosr2](https://github.com/santosr2)
- **style**: Implement blank line rules between and inside blocks ([7d48aff](https://github.com/santosr2/TerraTidy/commit/7d48aff408288b50985723bd334207c6bff2d0cb)) by [@santosr2](https://github.com/santosr2)
- **test**: Make TestToAbsPath cross-platform for Windows ([f8c8a40](https://github.com/santosr2/TerraTidy/commit/f8c8a404598c7e333ce730dc3efe969bc7adf0c4)) by [@santosr2](https://github.com/santosr2)
- **release**: Add RELEASE_NOTES.md to gitignore to prevent dirty state ([c42f118](https://github.com/santosr2/TerraTidy/commit/c42f118db5fec84c012787d9b07ce31329707aa9)) by [@santosr2](https://github.com/santosr2)
- **release**: Improve release notes and tag alias workflow ([8e7d7cd](https://github.com/santosr2/TerraTidy/commit/8e7d7cdd22402f19c20f055940d269e182129aa5)) by [@santosr2](https://github.com/santosr2)

### Other

- Add bump-my-version to mise and update bumpversion config ([12e9a9f](https://github.com/santosr2/TerraTidy/commit/12e9a9fed782753fd610d1f4d8efb891ad221493)) by [@santosr2](https://github.com/santosr2)
- Bump version to 0.2.0-alpha.2 ([a4b88fc](https://github.com/santosr2/TerraTidy/commit/a4b88fcc90c9d3565f8c399a58d40bc98d705316)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha] - 2026-01-08

### Added

- **cli**: Add TFLint integration info to rules list command ([54a642f](https://github.com/santosr2/TerraTidy/commit/54a642f34021d78fe6e9a62292eee131e612b93a)) by [@santosr2](https://github.com/santosr2)
- **output**: Add GitHub Actions annotations output format ([508924c](https://github.com/santosr2/TerraTidy/commit/508924cf4a4d47639d3b7d1230f7afbec4128f63)) by [@santosr2](https://github.com/santosr2)
- Add performance optimizations and parallel execution ([3123266](https://github.com/santosr2/TerraTidy/commit/312326655decf68dd47d685807dceaa1b06a2ca4)) by [@santosr2](https://github.com/santosr2)
- Wire up output formatters and file cache ([22d2426](https://github.com/santosr2/TerraTidy/commit/22d24269f6580efafb2652b0f259d87a1d352cfb)) by [@santosr2](https://github.com/santosr2)
- **release**: Add version alias tags for stable and pre-releases ([c523929](https://github.com/santosr2/TerraTidy/commit/c523929728707e9b0b2ae9f81bb1c138367bca27)) by [@santosr2](https://github.com/santosr2)
- **output**: Add table format with color support ([592bc9c](https://github.com/santosr2/TerraTidy/commit/592bc9c6a30fc5d4cc123e2f726bd922f958aa3b)) by [@santosr2](https://github.com/santosr2)
- **style**: Add naming and file organization rules ([22f3a0f](https://github.com/santosr2/TerraTidy/commit/22f3a0f80058b65434776bcbf4e3a4dbf26a1722)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Update changelog and readme for GitHub Actions format ([9b35bce](https://github.com/santosr2/TerraTidy/commit/9b35bceefe5c0edc7e26fb1883a3ef3c1a536824)) by [@santosr2](https://github.com/santosr2)
- Update all documentation for accuracy ([7113ae8](https://github.com/santosr2/TerraTidy/commit/7113ae83a0114c57404d1e000f0569cf91dc6aef)) by [@santosr2](https://github.com/santosr2)
- Fix version references to v0.1.0 ([da24621](https://github.com/santosr2/TerraTidy/commit/da246210598f7c0084a659a1f304007ed4158f72)) by [@santosr2](https://github.com/santosr2)
- Add table format to output formats documentation ([96b250a](https://github.com/santosr2/TerraTidy/commit/96b250aa97137880048d944cd91cb63487b8a35a)) by [@santosr2](https://github.com/santosr2)

### Fixed

- **test**: Use --no-verify for git commits in tests ([c50c7c4](https://github.com/santosr2/TerraTidy/commit/c50c7c4f4b39aef3ef55a893b371c651d19bbbd6)) by [@santosr2](https://github.com/santosr2)
- **action**: Use correct terratidy check command and consolidate actions ([d2fbd98](https://github.com/santosr2/TerraTidy/commit/d2fbd98136e8353fd67d673cd843c45150bc37d8)) by [@santosr2](https://github.com/santosr2)
- Correct changelog.md symlink path ([46a2c09](https://github.com/santosr2/TerraTidy/commit/46a2c0917a7b55bd67749e908f72ca7ecbd9091c)) by [@santosr2](https://github.com/santosr2)
- **style**: Exclude comments when counting blank lines between blocks ([856224b](https://github.com/santosr2/TerraTidy/commit/856224b807d0a70b0ae50d6912896ac063536100)) by [@santosr2](https://github.com/santosr2)
- **version**: Use Go build info for version when ldflags not set ([cd91e0b](https://github.com/santosr2/TerraTidy/commit/cd91e0b954ddcd70da4e63d6738924b861e4f294)) by [@santosr2](https://github.com/santosr2)
- **action**: Correct SARIF file path for working directory and output redirection ([ddc9a1b](https://github.com/santosr2/TerraTidy/commit/ddc9a1b31c507419b10ae12d8d5a69709de71461)) by [@santosr2](https://github.com/santosr2)
- **action**: Separate stdout and stderr for JSON/SARIF output formats ([424e091](https://github.com/santosr2/TerraTidy/commit/424e091806cd06b7e75082b0228bb82f5fbc957e)) by [@santosr2](https://github.com/santosr2)
- **action**: Build from source when testing in terratidy repo ([96e5484](https://github.com/santosr2/TerraTidy/commit/96e54843d5b95b692c855d9a58ea94ac9ac8dfdb)) by [@santosr2](https://github.com/santosr2)
- **sarif**: Ensure line/column numbers are at least 1 per SARIF spec ([6dcf341](https://github.com/santosr2/TerraTidy/commit/6dcf3414423d21a32179d0b9b3580410b42d6e15)) by [@santosr2](https://github.com/santosr2)

### Other

- Update gitignore and mise configuration ([3d4cabc](https://github.com/santosr2/TerraTidy/commit/3d4cabc453cfb8fce22237da700c21d42bb2c923)) by [@santosr2](https://github.com/santosr2)

## [0.1.0] - 2025-12-22

### Added

- Initialize TerraTidy project foundation ([6c5d998](https://github.com/santosr2/TerraTidy/commit/6c5d9987f05ea4f8c91b57aa1d3582bf0d2ce60e)) by [@santosr2](https://github.com/santosr2)
- Add initial Fmt and Style engines structure ([77bdc49](https://github.com/santosr2/TerraTidy/commit/77bdc49382db15bf1a5f098e8c17fd39db63e0e7)) by [@santosr2](https://github.com/santosr2)
- Add initial Lint engine and output types ([36817f4](https://github.com/santosr2/TerraTidy/commit/36817f4e00d971b0a62379222517137cc82ad7c9)) by [@santosr2](https://github.com/santosr2)
- Add comprehensive tooling and documentation ([5559cab](https://github.com/santosr2/TerraTidy/commit/5559cab1cfd14b3eb471274376d7aa3848fe07d0)) by [@santosr2](https://github.com/santosr2)
- Add tests and rename fmt package to format ([437e23c](https://github.com/santosr2/TerraTidy/commit/437e23c30cbc0f292d75bd4e1d590acc44e82eb4)) by [@santosr2](https://github.com/santosr2)
- Add hardcoded secrets detection and enhance CI ([b1b437a](https://github.com/santosr2/TerraTidy/commit/b1b437a3dbc7f1c92aef37e98e31c74e8bd08d7d)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Add LSP client integration ([58478f3](https://github.com/santosr2/TerraTidy/commit/58478f34f8a1296aa49f4e4e37167192ba7fc4f0)) by [@santosr2](https://github.com/santosr2)

### Changed

- Reduce complexity and re-enable revive rules ([ab2ba6f](https://github.com/santosr2/TerraTidy/commit/ab2ba6f6e22a336393b7c6f8e5968d030cb6b60f)) by [@santosr2](https://github.com/santosr2)
- Reduce complexity in style rules ([cb8a02b](https://github.com/santosr2/TerraTidy/commit/cb8a02bc31515009298dd9db09382da4286c1885)) by [@santosr2](https://github.com/santosr2)
- Reduce complexity in policy engine and style rules ([2efed2e](https://github.com/santosr2/TerraTidy/commit/2efed2e7080ff2c20f175a0b0b8d6653032e74ac)) by [@santosr2](https://github.com/santosr2)

### Documentation

- Add missing documentation pages for mkdocs site ([67bcc58](https://github.com/santosr2/TerraTidy/commit/67bcc588e23a7f176421c36eabc1e2ade67a17b7)) by [@santosr2](https://github.com/santosr2)
- Fix key feautures list ([2c5e492](https://github.com/santosr2/TerraTidy/commit/2c5e492d4c95eb5718e1081a06eaf799209d5893)) by [@santosr2](https://github.com/santosr2)
- Update CHANGELOG for v0.1.0 release ([f1834f1](https://github.com/santosr2/TerraTidy/commit/f1834f1148a97980ee93ded53438618b38596f95)) by [@santosr2](https://github.com/santosr2)

### Fixed

- Workflow version ([7b6b7a1](https://github.com/santosr2/TerraTidy/commit/7b6b7a1365cad8ccb4c1e0a938b306a6b49cd846)) by [@santosr2](https://github.com/santosr2)
- Update policy engine to OPA v1 Rego syntax ([889a962](https://github.com/santosr2/TerraTidy/commit/889a9624e66d5abe5c245fcddf40cf847f4179aa)) by [@santosr2](https://github.com/santosr2)
- Resolve revive linter warnings ([18bd3a7](https://github.com/santosr2/TerraTidy/commit/18bd3a768b3941cca91442adf3c1b34204ecb212)) by [@santosr2](https://github.com/santosr2)
- **ci**: Resolve Windows test failure with coverage path ([6b61bd7](https://github.com/santosr2/TerraTidy/commit/6b61bd7469aa2453179aca6a598dd0358a70e735)) by [@santosr2](https://github.com/santosr2)
- **ci**: Run coverage only on Ubuntu to avoid Windows path issues ([0db91ed](https://github.com/santosr2/TerraTidy/commit/0db91edf2ce07e744fce41036462a41437ceee18)) by [@santosr2](https://github.com/santosr2)
- **test**: Make groupFilesByDirectory tests platform-agnostic ([6353501](https://github.com/santosr2/TerraTidy/commit/63535011ea7af30d8ad52082cf5d4d4c35366c29)) by [@santosr2](https://github.com/santosr2)
- **test**: Make tests platform-agnostic for Windows CI ([9bf1ce3](https://github.com/santosr2/TerraTidy/commit/9bf1ce3943f1fd96ab1ec8d47f04d373207fa5f5)) by [@santosr2](https://github.com/santosr2)
- **release**: Update goreleaser config for v2 compatibility ([69173d4](https://github.com/santosr2/TerraTidy/commit/69173d46c047ab15152f5eb176e4d8b3c802b10b)) by [@santosr2](https://github.com/santosr2)
- **docker**: Simplify Dockerfile for goreleaser compatibility ([180d3b9](https://github.com/santosr2/TerraTidy/commit/180d3b9ff5314c5855e9f9fd2f28ace74b55819a)) by [@santosr2](https://github.com/santosr2)
- **ci**: Add Docker login for ghcr.io in release workflow ([bd7ff2b](https://github.com/santosr2/TerraTidy/commit/bd7ff2bdb8ae2f439147f9721d5ea21ffcad0e2a)) by [@santosr2](https://github.com/santosr2)
- **release**: Use TerraTidy repo for homebrew formula ([5b215c0](https://github.com/santosr2/TerraTidy/commit/5b215c051d3bbd5f43a24a12953e6b5333b7a0d5)) by [@santosr2](https://github.com/santosr2)
- **release**: Use github-native changelog format for proper attribution ([98e19e7](https://github.com/santosr2/TerraTidy/commit/98e19e70c016a518384f2445d69ae51d2eaf9417)) by [@santosr2](https://github.com/santosr2)

### Other

- Add assets ([3548d87](https://github.com/santosr2/TerraTidy/commit/3548d877ef4d20ee99c8f709d0b61d09f671cfa1)) by [@santosr2](https://github.com/santosr2)
- Sync everywhere Go to 1.25 ([d37124b](https://github.com/santosr2/TerraTidy/commit/d37124b5a475decf29cfbeb33526b477e31c2982)) by [@santosr2](https://github.com/santosr2)
- Fix pre-commit issues and update OPA import ([ca09160](https://github.com/santosr2/TerraTidy/commit/ca0916053469b0ad9e1c233659b138c273486868)) by [@santosr2](https://github.com/santosr2)
- Exclude test files from complexity rules in revive ([2564974](https://github.com/santosr2/TerraTidy/commit/2564974b584115610107da2427e6fc292a4c3de3)) by [@santosr2](https://github.com/santosr2)
- **vscode**: Add .gitignore for build artifacts ([854a95f](https://github.com/santosr2/TerraTidy/commit/854a95f9e3e8cadc990d3f4e93a9ae4f65ff0d20)) by [@santosr2](https://github.com/santosr2)
[Unreleased]: https://github.com/santosr2/TerraTidy/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha.4...v0.2.0
[0.2.0-alpha.4]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha.3...v0.2.0-alpha.4
[0.2.0-alpha.3]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha.2...v0.2.0-alpha.3
[0.2.0-alpha.2]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha...v0.2.0-alpha.2
[0.2.0-alpha]: https://github.com/santosr2/TerraTidy/compare/v0.1.0...v0.2.0-alpha
