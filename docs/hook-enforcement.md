# Git Hook Enforcement Documentation

This document describes the comprehensive git hook security system that prevents bypassing Go formatting and linting checks.

## 🛡️ Security Overview

Our project enforces **mandatory** code quality checks that **cannot be bypassed** using `--no-verify` or other common bypass methods. This ensures:

- ✅ All Go code is properly formatted (`go fmt`)
- ✅ All code passes static analysis (`go vet`, `golangci-lint`)
- ✅ Dependencies are properly managed (`go mod tidy`)
- ✅ Commit messages follow conventional format
- ✅ Security vulnerabilities are detected (`osv-scanner`)

## 🔒 Security Measures

### 1. Enhanced Pre-Commit Hook

Location: `.git/hooks/pre-commit`

**Bypass Protection:**
- Detects `GIT_NO_VERIFY=1` environment variable
- Detects `--no-verify` flag attempts
- Checks for common bypass environment variables
- Blocks commits with security error messages

**Quality Checks:**
- `go mod tidy` (with auto-staging)
- `go fmt` (blocks commit if files need formatting)
- `go vet` (blocks commit on analysis failures)
- `golangci-lint` (comprehensive linting)
- `osv-scanner` (security vulnerability scanning)
- `markdownlint` (documentation quality)
- Version auto-bumping
- GitHub workflow validation

### 2. Prepare-Commit-MSG Hook

Location: `.git/hooks/prepare-commit-msg`

**Fallback Protection:**
- Runs even when pre-commit hooks are bypassed
- Performs critical formatting checks
- Runs `go vet` as safety net
- Adds warning annotations to bypassed commits

### 3. Secure Commit Script

Location: `scripts/secure-commit.sh`

**Comprehensive Validation:**
- Validates conventional commit message format
- Runs all pre-commit checks programmatically
- Cannot be bypassed (doesn't use `--no-verify`)
- Provides clear error messages and guidance

### 4. Makefile Integration

**Secure Commit Targets:**
```bash
make commit MESSAGE="feat(api): add new endpoint"
make commit-staged MESSAGE="fix(db): resolve connection issue"
make commit-all MESSAGE="docs: update API documentation"
```

**Hook Status and Testing:**
```bash
make hooks-status      # Check security status
make validate-hooks    # Test bypass protection
```

## 📖 Usage Guidelines

### ✅ Recommended Commit Methods

1. **Using Makefile (Recommended):**
   ```bash
   # Commit staged changes
   make commit MESSAGE="feat(api): add user authentication"
   
   # Stage and commit all changes
   make commit-all MESSAGE="fix(db): resolve connection pool"
   ```

2. **Using Secure Script Directly:**
   ```bash
   ./scripts/secure-commit.sh "feat(auth): implement OAuth2 flow"
   ```

3. **Standard Git (with hooks active):**
   ```bash
   git add .
   git commit -m "feat(api): add new endpoint"
   ```

### ❌ Blocked Methods

These methods are **automatically blocked** by our security system:

```bash
# These will FAIL with security errors:
git commit --no-verify -m "message"
GIT_NO_VERIFY=1 git commit -m "message"
SKIP_HOOKS=1 git commit -m "message"
```

## 🧪 Testing Hook Security

Verify that bypass protection is working:

```bash
make validate-hooks
```

This command:
1. Creates a test commit
2. Attempts to bypass hooks
3. Verifies the bypass is blocked
4. Cleans up test artifacts

## 🚨 Emergency Procedures

If you absolutely must make an emergency commit (production outage, etc.):

1. **First, try to fix the issue properly:**
   ```bash
   make fmt        # Fix formatting
   make lint       # Fix linting issues
   make vet        # Fix static analysis issues
   ```

2. **For true emergencies:**
   - Contact a repository maintainer
   - Document the emergency in the commit message
   - Fix issues in a follow-up commit immediately

## 🔧 Configuration

### Environment Variables Blocked

The system blocks these bypass attempts:
- `GIT_NO_VERIFY=1`
- `SKIP_HOOKS=1`
- `SKIP_PRE_COMMIT=1`
- `NO_HOOKS=1`
- `BYPASS_HOOKS=1`
- `HUSKY_SKIP_HOOKS=1`
- `HUSKY_SKIP_INSTALL=1`

### Commit Message Format

All commits must follow [Conventional Commits](https://conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Examples:**
- `feat(api): add user authentication endpoint`
- `fix(db): resolve connection pool exhaustion`
- `docs: update installation instructions`
- `test(integration): add user repository tests`

Run `make commit-msg-help` for detailed format guidelines.

## 🏗️ Setup Instructions

### Initial Setup

```bash
make setup
```

This command:
- Installs dependencies
- Sets up git hooks
- Configures commit linting
- Enables security measures

### Verify Setup

```bash
make hooks-status
```

Expected output:
```
🔍 Git Hooks Security Status
================================
📋 Pre-commit hook: ✅ ACTIVE
📋 Prepare-commit-msg hook: ✅ ACTIVE
📋 Secure commit script: ✅ AVAILABLE
📋 Commitlint config: ✅ CONFIGURED

🛡️  Hook Bypass Protection:
  • --no-verify detection: ENABLED
  • Environment variable checking: ENABLED
  • Fallback formatting checks: ENABLED
  • Mandatory linting enforcement: ENABLED
```

## 🐛 Troubleshooting

### Common Issues

1. **"go fmt" issues:**
   ```bash
   make fmt  # Fix formatting issues
   ```

2. **"golangci-lint" failures:**
   ```bash
   make lint  # See specific linting issues
   ```

3. **"go vet" problems:**
   ```bash
   make vet  # See static analysis issues
   ```

4. **Commit message format errors:**
   ```bash
   make commit-msg-help  # See format guidelines
   ```

### Hook Not Working

If hooks aren't running:

```bash
# Reinstall hooks
make setup

# Check permissions
chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/prepare-commit-msg
chmod +x scripts/secure-commit.sh

# Verify status
make hooks-status
```

### CI/CD Integration

For continuous integration environments, hooks run automatically. The system is designed to work in CI environments without additional configuration.

## 📊 Security Benefits

1. **Code Quality:** Ensures consistent formatting and style across the codebase
2. **Security:** Prevents vulnerable code from being committed
3. **Maintainability:** Enforces best practices and catches issues early
4. **Team Standards:** Ensures all team members follow the same conventions
5. **Automation:** Reduces manual code review time by catching issues automatically

## 📝 Maintenance

### Updating Hooks

When updating hook scripts:

1. Make changes to the hook files
2. Test with `make validate-hooks`
3. Document changes in this file
4. Notify team members of updates

### Adding New Checks

To add new quality checks:

1. Edit `.git/hooks/pre-commit`
2. Add corresponding fallback to `.git/hooks/prepare-commit-msg`
3. Update `scripts/secure-commit.sh` if needed
4. Test thoroughly with `make validate-hooks`
5. Update this documentation

Remember: Any new checks should be **mandatory** and **non-bypassable** to maintain security integrity.