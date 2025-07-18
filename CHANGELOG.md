# Changelog

All notable changes to EntityStore will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.5] - 2025-01-25

### Changed
- **Simplified GSI Configuration**: Removed GSI2 and GSI3 configurations to match production setup
  - Now only supports single GSI (GSI1) with physical keys PK1 and SK1
  - Reduces operational costs by maintaining only one GSI
  - Aligns with production table schema

### Updated
- **DynamoDB Setup Scripts**: Enhanced to support pay-per-request billing mode
  - Pay-per-request is now the default billing mode
  - Scripts support both pay-per-request and provisioned modes
  - Added production table example in setup documentation

### Documentation
- Updated setup guides to reflect single GSI configuration
- Added production table example with pay-per-request billing
- Clarified that only GSI1 is supported with PK1/SK1 physical keys
- Enhanced setup script documentation with billing mode options

## [0.2.2] - 2025-01-19

### Added
- **GSI Key Mapping**: Automatic mapping between logical GSI names (GSI1PK/GSI1SK) and physical DynamoDB attributes (PK1/SK1)
  - Added `GSIConfig` to handle GSI key name mappings
  - Put method now correctly maps GSI keys when storing items
  - Query builder uses correct physical key names in queries
- **Just Build System**: Added Justfile for streamlined development workflow
  - Common tasks like build, test, setup, and release
  - DynamoDB table setup and verification commands
- **Setup Documentation**: Comprehensive setup and prerequisites guides
  - `PREREQUISITES.md`: System requirements and AWS setup
  - `SETUP_GUIDE.md`: Step-by-step setup instructions
  - Shell scripts for DynamoDB table setup and verification
- **DynamoDB Setup Scripts**:
  - `setup-dynamodb-table.sh`: Creates properly configured DynamoDB table
  - `verify-dynamodb-setup.sh`: Verifies table structure and permissions

### Fixed
- GSI queries now use correct physical key names (PK1/SK1) instead of logical names
- Updated tests to reflect correct GSI key names

## [0.2.1] - 2025-01-19

### Fixed
- **Critical Bug Fix**: Query method now correctly uses the datastore's table name instead of the one from QueryParams
  - Fixed issue where GSI queries would fail with "ValidationException: Value '' at 'tableName' failed to satisfy constraint"
  - Both `Query` and `Stream` methods now properly use `d.tableName` from the datastore instance
  - Added deprecation notice to `QueryParams.TableName` field as it's no longer used
  - Added test coverage for the bug fix

### Changed
- Deprecated `QueryParams.TableName` field - the table name is now always taken from the DataStore instance

## [0.2.0] - 2025-01-19

### Added
- **GSI Query Optimization**: New fluent query builder for Global Secondary Index queries
  - `GSIQueryBuilder` with methods like `WithPartitionKey()`, `WithSortKey()`, `WithSortKeyPrefix()`
  - Convenience methods: `QueryByGSI1PK()`, `QueryByGSI1PKAndSKPrefix()`, `QueryByGSI1PKWithFilter()`
  - Support for sort key ranges with `WithSortKeyBetween()`, `WithSortKeyGreaterThan()`, etc.
  - Filter expression support with `WithFilter()`
  - Streaming support for GSI queries
  - Complete test coverage in `gsi_query_test.go`

- **Time-Based Query Patterns**: Specialized support for time-based queries
  - `TimeRangeQueryBuilder` for time-based access patterns
  - Convenience methods: `InLastHours()`, `InLastDays()`, `Today()`, `ThisWeek()`, `ThisMonth()`
  - Time range queries: `Between()`, `After()`, `Before()`
  - Sort order control: `Latest()` (newest first) and `Oldest()` (oldest first)
  - Time window iterator for processing large date ranges in chunks
  - New convenience methods: `QueryLatestItems()`, `QueryItemsSince()`, `StreamLatestItems()`
  - Time-based pagination support with `QueryWithTimePagination()`

- **Query Enhancements**
  - Added `ScanIndexForward` to `QueryParams` for controlling sort order
  - Updated `Query` and `Stream` implementations to support sort order
  - Enhanced streaming to respect sort order for time-based queries

### Documentation
- Added `GSI_OPTIMIZATION_GUIDE.md` with comprehensive GSI query patterns and best practices
- Added `ADVANCED_QUERY_PATTERNS.md` documenting future query enhancement roadmap
- Updated quick reference with GSI and time-based query examples
- Enhanced documentation for time-based access patterns
- Added time-based query best practices to GSI optimization guide

### Changed
- Updated `storagemodels.QueryParams` to include `ScanIndexForward` field
- Enhanced DynamoDB query implementation to support sort order control
- Improved version management to 0.2.0

## [0.1.0] - 2025-01-18

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

