# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0-alpha.4] - 2026-04-04

### Added

- **plugins**: Add YAML and Bash rule loaders with examples ([#4](https://github.com/santosr2/TerraTidy/pull/4)) by [@santosr2](https://github.com/santosr2)

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

### Ci

- **release**: Mark pre-releases as latest on GitHub ([8f2a130](https://github.com/santosr2/TerraTidy/commit/8f2a13026e21d4a64b4ecdebe3613ad8a6fe134a)) by [@santosr2](https://github.com/santosr2)
- Add dependabot and development utility scripts ([#5](https://github.com/santosr2/TerraTidy/pull/5)) by [@santosr2](https://github.com/santosr2)
- Add permissions, timeouts, concurrency, and path filters to workflows ([#35](https://github.com/santosr2/TerraTidy/pull/35)) by [@santosr2](https://github.com/santosr2)
- Fix injection risks, pin actions to SHAs, and harden Dockerfile ([#36](https://github.com/santosr2/TerraTidy/pull/36)) by [@santosr2](https://github.com/santosr2)
- Add security scanning, quality checks, and pin mise tool versions ([#38](https://github.com/santosr2/TerraTidy/pull/38)) by [@santosr2](https://github.com/santosr2)
- Add SBOM, cosign signing, attestations, and OpenSSF Scorecard ([#39](https://github.com/santosr2/TerraTidy/pull/39)) by [@santosr2](https://github.com/santosr2)
- Improve release pipeline with shared version parsing, crane, and smoke tests ([#40](https://github.com/santosr2/TerraTidy/pull/40)) by [@santosr2](https://github.com/santosr2)
- **quality**: Cache pre-commit hooks and skip branch guard in CI ([#45](https://github.com/santosr2/TerraTidy/pull/45)) by [@santosr2](https://github.com/santosr2)
- Add PR title checker and standardize commit conventions ([#51](https://github.com/santosr2/TerraTidy/pull/51)) by [@santosr2](https://github.com/santosr2)

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

### Documentation

- Update changelog for v0.2.0-alpha.2 ([96c06be](https://github.com/santosr2/TerraTidy/commit/96c06be3823d47ea0255eeee22d912939954ec6e)) by [@github-actions[bot]](https://github.com/github-actions[bot])
- Clarify that 'latest' Docker tag includes pre-releases ([e50ac4c](https://github.com/santosr2/TerraTidy/commit/e50ac4c84a473792adc8237b87c189d89b44936d)) by [@santosr2](https://github.com/santosr2)

### Fixed

- **release**: Improve changelog and add Docker alias tag updates ([3426e86](https://github.com/santosr2/TerraTidy/commit/3426e8606a125ff83e2e28afad24fc175988d44f)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve internal blank lines and re-run fmt after style fixes ([9e76798](https://github.com/santosr2/TerraTidy/commit/9e76798ab18ec6c381f602e0fdbedd7cded073e1)) by [@santosr2](https://github.com/santosr2)
- **cli**: Fix color flag and add changelog links ([b11560c](https://github.com/santosr2/TerraTidy/commit/b11560c1bab748e08cd634e98d31543257db4798)) by [@santosr2](https://github.com/santosr2)
- **cli**: Apply global color flag to all commands ([3f50fbb](https://github.com/santosr2/TerraTidy/commit/3f50fbb473810f03efe5e35ba683f6fa31575b41)) by [@santosr2](https://github.com/santosr2)
- **style**: Preserve inline comments when reordering HCL attributes ([#2](https://github.com/santosr2/TerraTidy/pull/2)) by [@santosr2](https://github.com/santosr2)

### Ci

- Add one-time workflow to fix Docker alias tags ([61c8f81](https://github.com/santosr2/TerraTidy/commit/61c8f811e744b392ace8bb11e6ed335b00df5f99)) by [@santosr2](https://github.com/santosr2)
- Remove one-time Docker fix workflow ([3ab9f8b](https://github.com/santosr2/TerraTidy/commit/3ab9f8b7dd0e29eaa76fb32cf8d1e6cdc858e512)) by [@santosr2](https://github.com/santosr2)
- Add workflow to fix Docker alias tags ([a7faff1](https://github.com/santosr2/TerraTidy/commit/a7faff11eeb60a6657d84b11aafbd9a17374df4d)) by [@santosr2](https://github.com/santosr2)
- Remove one-time Docker fix workflow ([366979c](https://github.com/santosr2/TerraTidy/commit/366979ca70693210230444924a1babe384dd5592)) by [@santosr2](https://github.com/santosr2)
- Add codecov configuration ([41234a3](https://github.com/santosr2/TerraTidy/commit/41234a3454df44181e7c20fbe4e37b5f78beae63)) by [@santosr2](https://github.com/santosr2)

## [0.2.0-alpha.2] - 2026-01-12

### Added

- **release**: Add automated changelog and fix pre-release handling ([6336002](https://github.com/santosr2/TerraTidy/commit/63360025c80b1c0b8324bb03d5680f7eba9ea762)) by [@santosr2](https://github.com/santosr2)
- Add bump-my-version for version management ([cfabfba](https://github.com/santosr2/TerraTidy/commit/cfabfba767be4383e18077220cf57c6258650e7d)) by [@santosr2](https://github.com/santosr2)

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

### Ci

- Add coverpkg flag for accurate cross-package coverage ([ac0c54c](https://github.com/santosr2/TerraTidy/commit/ac0c54cccb18a1c66ccc987f5c720529313a6e63)) by [@santosr2](https://github.com/santosr2)

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

[0.2.0-alpha.4]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha.3...v0.2.0-alpha.4
[0.2.0-alpha.3]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha.2...v0.2.0-alpha.3
[0.2.0-alpha.2]: https://github.com/santosr2/TerraTidy/compare/v0.2.0-alpha...v0.2.0-alpha.2
[0.2.0-alpha]: https://github.com/santosr2/TerraTidy/compare/v0.1.0...v0.2.0-alpha
