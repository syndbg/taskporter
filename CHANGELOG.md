# 1.0.0 (2025-07-20)


### Bug Fixes

* **ci:** golangci-lint and goenv ([eb45510](https://github.com/syndbg/taskporter/commit/eb455104e6d68b201a1c88ed5931547d8f76359a))
* **ci:** golangci-lint version ([7756ed4](https://github.com/syndbg/taskporter/commit/7756ed4a6d585bc310a4d92179d609dfbf7e7071))
* **ci:** golangci-lint version string ([5a1d9a8](https://github.com/syndbg/taskporter/commit/5a1d9a864137bc31dfcfdd9fa73c34a6735c5ce6))
* **ci:** golangci-lint-action  version ([99ad2b1](https://github.com/syndbg/taskporter/commit/99ad2b1f80ccb6b7ffc0b430c5fc6feaa3b8166a))
* **ci:** lint offencs ([742a9f0](https://github.com/syndbg/taskporter/commit/742a9f075fdb51cffa1091066a3fadf3a5936764))
* **ci:** use latest commit ([d6540f4](https://github.com/syndbg/taskporter/commit/d6540f4d2e484659ed80919a9f53b485a66010c5))
* **ci:** use recommended caching ([7584dbf](https://github.com/syndbg/taskporter/commit/7584dbf350f71e604b3d102fe7aed03074de4950))
* consistently name the test and implementation files ([aded199](https://github.com/syndbg/taskporter/commit/aded199b6e5c27875fe5d2d06b85defa52150131))
* correct gosec import path in CI workflow ([96aaea3](https://github.com/syndbg/taskporter/commit/96aaea310a14874a144caecdd9040e184fcefbf5))
* integrate GoReleaser with semantic-release for proper binary releases ([8568dc2](https://github.com/syndbg/taskporter/commit/8568dc2c34fff48845f496629dbfdff8441fb9a1))
* lowercase error strings to follow Go conventions ([3f3195e](https://github.com/syndbg/taskporter/commit/3f3195e7f94d25a059e12cc230e3f3adfc3fa5f7))
* resolve all wsl_v5 whitespace linting issues ([b78b626](https://github.com/syndbg/taskporter/commit/b78b626d11e3bcfda3e4c09f68cdbc5a164f46cc))
* resolve linter errors in security and run commands ([803add6](https://github.com/syndbg/taskporter/commit/803add646ed5461ab9d7f6e809d2b815b698bbb1))
* Update .gitignore to allow taskporter- prefixed files ([6f0a8b5](https://github.com/syndbg/taskporter/commit/6f0a8b50d8e2877c2e454fa8dbc7216480fa3595))
* update GitHub Actions to use latest versions ([26fe321](https://github.com/syndbg/taskporter/commit/26fe32169045b2a1d0caec7bbc7352ecc3d853e2))
* update golangci-lint config for v8 action compatibility ([7e92d28](https://github.com/syndbg/taskporter/commit/7e92d280a9de7656fef19f37f3bf0d31a6039477))
* update golangci-lint to latest version in CI ([0e351d3](https://github.com/syndbg/taskporter/commit/0e351d33ff913bc96197f39cd7a90a98a97b9eb8))


### Features

* Add --no-interactive flag to disable interactive mode ([282b7d3](https://github.com/syndbg/taskporter/commit/282b7d3875cb3e8b0a59791bb82ac7461afb4453))
* Add comprehensive shell completion support ([038def1](https://github.com/syndbg/taskporter/commit/038def1ec7fa2b17959de62d1afe24ad9a884bcd))
* Add comprehensive tests for VSCode to JetBrains converter ([e7dc4f1](https://github.com/syndbg/taskporter/commit/e7dc4f188c4631c43e9676df7471f1bb30730bd1))
* add golangci-lint config with gosec disabled ([1828435](https://github.com/syndbg/taskporter/commit/18284353509b3f2317c3113bb06e55c4dcfcba13))
* Add interactive task selection mode using Bubble Tea ([a5910ec](https://github.com/syndbg/taskporter/commit/a5910ecfc96fdda60e1d7e2ae088d9aa0a5c9569))
* Add JetBrains IDE support with Application and Gradle run configurations ([868186f](https://github.com/syndbg/taskporter/commit/868186f36c767b812f33048ce2d0bf3d0ff5e29d))
* add semantic-release for automatic versioning and releases ([19410c6](https://github.com/syndbg/taskporter/commit/19410c692fc0d9c92e243de786ba5f0928968b7e))
* bump Go version to 1.24.5 across all configs ([99a30e8](https://github.com/syndbg/taskporter/commit/99a30e8e6ba7f955c8707ae0b6b61a0c40449a27))
* implement comprehensive configuration migration system ([8727226](https://github.com/syndbg/taskporter/commit/8727226895a03c34e8b3efda549c551c24541d96))
* implement comprehensive security sanitization system ([0cbb1a2](https://github.com/syndbg/taskporter/commit/0cbb1a2bcc04d684bb521094bade4d338a2549ec))
* implement preLaunchTask execution support ([2eac2d6](https://github.com/syndbg/taskporter/commit/2eac2d692c1d6abb6016cf77830bb6857035ffc6))
* Implement VSCode tasks → JetBrains conversion ([e0bed17](https://github.com/syndbg/taskporter/commit/e0bed176b17d010e0de40c6046d3310aa2a725d7))
* Phase 1 complete - Foundation & CI/CD setup ([c4a61e4](https://github.com/syndbg/taskporter/commit/c4a61e4109bed6b37dd5cf70e12a1b350ffa0370))
* Phase 2.1 complete - VSCode tasks integration ([e340d03](https://github.com/syndbg/taskporter/commit/e340d0303278d51e54dc74947a128963c3bc7d99))
* Phase 2.2 complete - Task execution implementation ([fc78cae](https://github.com/syndbg/taskporter/commit/fc78cae168f9222f6900b681736eb5ea3c6625e7))
* show .vscode and .idea files as they're going to be both supported ([61ebb0e](https://github.com/syndbg/taskporter/commit/61ebb0e68f88fae8bf3b17ea1b674095b9a3f36f))
* use official gosec GitHub Action for security scanning ([77abfcf](https://github.com/syndbg/taskporter/commit/77abfcf3f177ce12d1360f3083b0d69d1dc3cb8a))

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is automatically generated by [semantic-release](https://github.com/semantic-release/semantic-release).

## [Unreleased]

### Added
- Automatic semantic versioning and releases
- Conventional commit validation
- Automatic changelog generation

### Changed
- CI pipeline now uses semantic-release for automated releases
- Simplified CI by using only golangci-lint for linting and security

### Fixed
- Resolved all wsl_v5 whitespace linting issues
- Updated golangci-lint configuration for compatibility with latest versions
