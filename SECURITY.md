# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability within Nexus (gore ORM), please follow responsible disclosure:

1. **Do NOT** create a public GitHub issue for security vulnerabilities.
2. Send details to the project maintainers via:
   - Private security report through GitHub's [Security Advisories](https://github.com/hexiefeng/Nexus/security/advisories/new)
   - Or email: security@example.com (replace with actual contact)

3. Please include the following information:
   - Type of vulnerability
   - Full paths of source file(s) related to the vulnerability
   - Location of the affected source code
   - Any special configuration required to reproduce the issue
   - Step-by-step instructions to reproduce the issue
   - Proof-of-concept or exploit code (if possible)
   - Impact of the issue including how an attacker might exploit it

4. Expected response time:
   - Acknowledgment: within 48 hours
   - Initial assessment: within 7 days
   - Fix timeline: depends on severity (critical: 7 days, high: 30 days, medium/low: next release)

## Security Best Practices

When using gore ORM:

1. **Never** concatenate user input directly into SQL strings - always use parameterized queries via the Query Builder
2. **Never** expose database credentials in source code - use environment variables
3. **Always** validate entity data before calling `SaveChanges()`
4. **Enable** TLS/SSL in production database connections

## Security Checklist for Production

- [ ] Use environment variables for all credentials
- [ ] Enable SSL/TLS for database connections
- [ ] Run `go mod verify` to verify dependencies
- [ ] Run `govulncheck ./...` before deployments
- [ ] Review CODEOWNERS for proper access controls
