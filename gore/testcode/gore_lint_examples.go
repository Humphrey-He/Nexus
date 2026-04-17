package testcode

// ===============================================================================
// gore-lint CLI Usage Examples
// ===============================================================================

// Gore-lint is a static analysis tool for SQL index recommendations.
//
// This file contains examples of how to use gore-lint in various scenarios.

// ===============================================================================
// Example 1: Basic Usage with Schema JSON
// ===============================================================================

// Basic command to analyze Go source code with a schema file:
//
//   gore-lint check --schema ./schema.json ./...
//
// This will:
// 1. Load the schema from schema.json
// 2. Parse all Go files in ./...
// 3. Extract query patterns from DbSet usage
// 4. Analyze against the schema
// 5. Output suggestions in JSON format

// ===============================================================================
// Example 2: MySQL Database Analysis
// ===============================================================================

// Connect to MySQL database directly:
//
//   gore-lint check --dsn "mysql://user:pass@localhost:3306/gore_test" --db-type mysql ./...
//
// This will:
// 1. Connect to MySQL database
// 2. Query INFORMATION_SCHEMA for table/index metadata
// 3. Analyze Go source code
// 4. Report index recommendations

// ===============================================================================
// Example 3: PostgreSQL Database Analysis
// ===============================================================================

// Connect to PostgreSQL database:
//
//   gore-lint check --dsn "postgres://user:pass@localhost:5432/gore_test" --db-type postgres ./...
//
// This will:
// 1. Connect to PostgreSQL database
// 2. Query pg_catalog for table/index metadata
// 3. Analyze Go source code
// 4. Report index recommendations

// ===============================================================================
// Example 4: Output Formats
// ===============================================================================

// Output in JSON format (default):
//
//   gore-lint check --schema ./schema.json --format json ./...
//
// Output in text format:
//
//   gore-lint check --schema ./schema.json --format text ./...
//
// Text output is more readable for human inspection.

// ===============================================================================
// Example 5: Schema JSON Structure
// ===============================================================================

// The schema JSON file contains database table and index information:
//
//   {
//     "tables": [
//       {
//         "tableName": "users",
//         "columns": [
//           {"name": "id", "type": "int"},
//           {"name": "name", "type": "varchar"},
//           {"name": "email", "type": "varchar"},
//           {"name": "status", "type": "tinyint"},
//           {"name": "created_at", "type": "timestamp"}
//         ],
//         "indexes": [
//           {"name": "PRIMARY", "columns": ["id"], "unique": true, "method": "btree"},
//           {"name": "idx_name", "columns": ["name"], "unique": false, "method": "btree"},
//           {"name": "idx_status", "columns": ["status"], "unique": false, "method": "btree", "isVisible": false}
//         ]
//       }
//     ]
//   }
//
// MySQL 8.0+ specific fields:
// - isVisible: true/false for invisible indexes
// - collation: "A" (ascending) or "D" (descending)

// ===============================================================================
// Example 6: Index Recommendation Rules
// ===============================================================================

// gore-lint implements the following index analysis rules:
//
// IDX-001: Leftmost Match Validation
//   - Checks if composite indexes are used with leftmost column
//   - Example: Index (a, b, c) requires WHERE a = ? to use the index
//
// IDX-002: Function Index Detection
//   - Detects queries using functions on indexed columns
//   - Example: WHERE LOWER(name) = ? prevents index usage
//
// IDX-003: Type Mismatch
//   - Detects implicit type conversions
//   - Example: WHERE age = '25' (string vs int) prevents index usage
//
// IDX-004: LIKE Prefix Wildcard
//   - Detects LIKE patterns starting with %
//   - Example: WHERE name LIKE '%Alice%' cannot use index
//
// IDX-005: Negation Detection
//   - Detects !=, NOT IN, IS NOT NULL conditions
//   - These typically cannot use indexes efficiently
//
// IDX-006: Missing Index
//   - Suggests indexes for frequently queried columns
//
// IDX-007: Redundant Index
//   - Detects duplicate or overlapping indexes
//
// IDX-008: ORDER BY Index
//   - Checks if ORDER BY columns have indexes
//
// IDX-009: JOIN Index
//   - Checks if JOIN columns have indexes
//
// IDX-010: Low Selectivity Index
//   - Detects indexes on low-cardinality columns
//
// MySQL-specific rules:
//
// IDX-MYSQL-001: Invisible Index
//   - Detects invisible indexes that may be unused
//
// IDX-MYSQL-002: Descending Index
//   - Validates ORDER BY direction matches index collation
//
// IDX-MYSQL-003: Hash Index Range
//   - Detects range queries on HASH indexes (which don't support ranges)

// ===============================================================================
// Example 7: CI/CD Integration
// ===============================================================================

// Integrate gore-lint into GitHub Actions:
//
//   name: CI
//   on: [push, pull_request]
//   jobs:
//     gore-lint:
//       runs-on: ubuntu-latest
//       steps:
//         - uses: actions/checkout@v4
//         - name: Run gore-lint
//           run: |
//             curl -sL https://example.com/gore-lint | sh
//             gore-lint check --schema ./schema.json ./...
//         - name: Upload results
//           uses: actions/upload-artifact@v4
//           with:
//             name: gore-lint-results
//             path: gore-lint-results.json

// ===============================================================================
// Example 8: Query Patterns Detection
// ===============================================================================

// gore-lint detects these query patterns in Go code:
//
// Query building with WhereField:
//   api.Set[User](ctx).Query().From("users").WhereField("name", "=", "Alice")
//   - Detected: table=users, field=name, operator==, value=Alice
//
// Query building with WhereIn:
//   api.Set[User](ctx).Query().From("users").WhereIn("id", 1, 2, 3)
//   - Detected: table=users, field=id, operator=IN, values=[1,2,3]
//
// Query building with WhereLike:
//   api.Set[User](ctx).Query().From("users").WhereLike("name", "%Alice%")
//   - Detected: table=users, field=name, operator=LIKE, value=%Alice%
//
// Query building with OrderBy:
//   api.Set[User](ctx).Query().From("users").OrderBy("created_at DESC")
//   - Detected: table=users, orderBy=[{field:created_at, direction:DESC}]

// ===============================================================================
// Example 9: Recommendations Output
// ===============================================================================

// Example JSON output:
//
//   {
//     "version": "0.1",
//     "generatedAt": "2026-04-17T10:30:00Z",
//     "target": "./internal/repository",
//     "suggestions": [
//       {
//         "ruleId": "IDX-001",
//         "severity": 1,
//         "message": "联合索引 users 的首列 name 未在 WHERE 中出现",
//         "reason": "B-Tree 索引遵循最左匹配原则，必须从第一列开始",
//         "recommendation": "考虑在 WHERE 中加入 name 条件或调整索引顺序",
//         "confidence": 0.95,
//         "sourceFile": "internal/repository/user.go",
//         "lineNumber": 42,
//         "tags": ["index", "leftmost-match"]
//       }
//     ],
//     "stats": {
//       "total": 1,
//       "info": 0,
//       "warn": 1,
//       "high": 0,
//       "critical": 0
//     }
//   }
//
// Severity levels:
//   - 0: Info (informational)
//   - 1: Warn (warning)
//   - 2: High (high priority)
//   - 3: Critical (critical issue)

// ===============================================================================
// Example 10: Exit Codes
// ===============================================================================

// gore-lint returns the following exit codes:
//
//   0: Success, no issues found
//   1: Success, issues found (suggestions)
//   2: Error (invalid arguments, file not found, etc.)
//
// Use exit code 1 in CI to fail the build when index issues are found:
//
//   gore-lint check --schema ./schema.json ./...
//   if [ $? -eq 1 ]; then
//     echo "Index recommendations found!"
//     # Treat as warning or failure depending on requirements
//   fi
