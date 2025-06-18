# Changelog

All notable changes to EntityStore will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-06-18

### Added
- Enhanced streaming API with single-channel design
  - Configurable buffer sizes and retry logic
  - Progress tracking with callbacks
  - Per-item metadata (index, page number, timestamp)
  - Built-in retry for transient errors
- Semantic error types in new `errors` package
  - `NotFound`, `AlreadyExists`, `ValidationError`, `ConditionFailed`
  - Helper functions for error checking
  - Full compatibility with `errors.Is()`
- Type-safe storage with `MultiTypeStorage`
  - Compile-time type checking
  - No runtime type assertions needed
  - Support for multiple types with same key
- Comprehensive mock implementation in `datastore/mock`
  - Thread-safe in-memory storage
  - Configurable error injection
  - Custom query/stream behavior
  - Helper methods for testing
- Test automation infrastructure
  - Makefile with common tasks
  - GitHub Actions CI/CD workflows
  - Integration tests with DynamoDB Local
  - Code coverage reporting with Codecov
  - golangci-lint configuration
- Package-level documentation for all packages
- Migration guide for adopting type-safe storage

### Changed
- Consolidated `QueryParams` and `StreamQueryParams` into single type
- Improved thread safety in `StorageManager` with proper mutex usage
- Updated `GetOne` to return `NotFound` error instead of `nil, nil`
- Enhanced error messages with more context

### Fixed
- Thread safety issues in storage manager
- Race conditions in concurrent operations
- Error handling inconsistencies

### Security
- Added security scanning in CI pipeline
- No hardcoded credentials in code

