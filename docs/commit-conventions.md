# Commit Message Conventions

This project follows the [Conventional Commits](https://conventionalcommits.org/) specification to ensure consistent and meaningful commit messages.

## Quick Reference

### Format
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Examples
```bash
feat(api): add user authentication endpoint
fix(db): resolve connection pool exhaustion  
docs: update API documentation
test(integration): add user repository tests
ci: add commitlint workflow
```

## Commit Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | ✨ A new feature | `feat(auth): add OAuth2 login` |
| `fix` | 🐛 A bug fix | `fix(api): handle null pointer in user handler` |
| `docs` | 📚 Documentation only changes | `docs: update installation guide` |
| `style` | 💄 Code style changes (formatting, etc) | `style: fix indentation in handlers` |
| `refactor` | ♻️ Code refactoring without feature changes | `refactor(db): simplify query builder` |
| `perf` | ⚡ Performance improvements | `perf(api): optimize database queries` |
| `test` | 🧪 Adding missing tests | `test(unit): add user service tests` |
| `build` | 📦 Changes to build system or dependencies | `build: update go version to 1.21` |
| `ci` | 👷 Changes to CI configuration | `ci: add integration test workflow` |
| `chore` | 🔧 Other changes that don't modify src/test | `chore: update .gitignore` |
| `revert` | ⏪ Reverts a previous commit | `revert: remove broken feature` |

## Scopes

Scopes provide additional context about what part of the codebase is affected:

### API & Handlers
- `api` - General API changes
- `handlers` - HTTP request handlers
- `middleware` - HTTP middleware
- `auth` - Authentication/authorization
- `validation` - Input validation

### Database & Repository
- `db` - Database operations
- `repository` - Repository layer
- `migrations` - Database migrations
- `collections` - PocketBase collections

### Domain & Business Logic  
- `domain` - Domain models and logic
- `services` - Service layer
- `models` - Data models

### Infrastructure
- `config` - Configuration management
- `container` - Dependency injection
- `logging` - Logging functionality
- `monitoring` - Monitoring and metrics

### Testing
- `tests` - General test changes
- `integration` - Integration tests
- `unit` - Unit tests  
- `e2e` - End-to-end tests
- `mocks` - Test mocks and stubs

### DevOps & CI
- `ci` - Continuous integration
- `docker` - Docker configuration
- `deployment` - Deployment scripts
- `scripts` - Build and utility scripts

### Documentation
- `docs` - Documentation files
- `readme` - README updates
- `changelog` - Changelog updates

### Project Specific
- `tasks` - Task-related functionality
- `projects` - Project management features
- `users` - User management
- `comments` - Comment system
- `tags` - Tag functionality

## Tools and Enforcement

### Local Development

#### Installation
```bash
# Install dependencies
npm install

# Set up git hooks
npm run prepare
```

#### Manual Validation
```bash
# Lint last commit
make commit-lint

# Lint commit range
make commit-lint-range FROM=abc123 TO=def456

# Lint all commits in branch
make commit-lint-branch

# Show format help
make commit-msg-help
```

### Git Hooks

The project uses [Husky](https://typicode.github.io/husky/) to enforce commit conventions:

- **commit-msg hook**: Validates commit message format
- **pre-commit hook**: Runs integration test compilation check

### CI/CD Integration

GitHub Actions automatically validates commit messages on:
- Push to `develop`, `release`, `main` branches
- Pull requests to those branches
- PR title validation (must follow conventional format)

## Writing Good Commit Messages

### 1. Use the Imperative Mood
Write commit messages as if you're giving a command:
```bash
✅ feat(api): add user authentication endpoint
❌ feat(api): added user authentication endpoint
❌ feat(api): adds user authentication endpoint
```

### 2. Be Specific and Descriptive
```bash
✅ fix(auth): handle expired JWT tokens gracefully
❌ fix: auth bug
```

### 3. Keep Subject Lines Under 100 Characters
```bash
✅ feat(api): add comprehensive user authentication with JWT and refresh tokens
❌ feat(api): add comprehensive user authentication system with JWT tokens and refresh token functionality for secure login
```

### 4. Use Body for Complex Changes
```bash
feat(auth): implement OAuth2 authentication

Add support for Google and GitHub OAuth2 providers.
Includes token refresh mechanism and user profile sync.

Closes #123
```

### 5. Reference Issues and PRs
```bash
fix(db): resolve connection pool exhaustion

The connection pool was not properly releasing connections
after failed transactions, leading to pool exhaustion.

Fixes #456
Closes #789
```

## Examples by Category

### Features
```bash
feat(api): add user registration endpoint
feat(auth): implement JWT token refresh
feat(tasks): add task priority levels
feat(projects): support project templates
```

### Bug Fixes
```bash
fix(db): prevent deadlock in concurrent updates
fix(api): validate required fields in user creation
fix(auth): handle case-insensitive email lookup
fix(middleware): log request correlation IDs
```

### Documentation
```bash
docs: add API endpoint documentation
docs(auth): document OAuth2 flow
docs: update deployment instructions
docs(db): add migration guide
```

### Tests
```bash
test(integration): add user repository tests
test(unit): improve auth service coverage
test(e2e): add login flow validation
test: fix flaky concurrency tests
```

### Refactoring
```bash
refactor(api): extract common response patterns
refactor(db): simplify query builder interface
refactor(auth): consolidate token validation logic
refactor: remove unused dependencies
```

### CI/CD
```bash
ci: add integration test workflow
ci: improve Docker build performance
ci(security): add dependency vulnerability scanning
ci: fix coverage reporting threshold
```

## Breaking Changes

For breaking changes, add `!` after the type/scope and explain in the footer:

```bash
feat(api)!: change user ID format to UUID

BREAKING CHANGE: User IDs are now UUIDs instead of integers.
This affects all API endpoints that accept user IDs.

Migration guide available in docs/migration-v2.md
```

## Troubleshooting

### Common Issues

1. **"subject may not be empty"**
   - Ensure there's a colon and space after the type: `feat: description`

2. **"type may not be empty"**
   - Start with a valid type: `feat`, `fix`, `docs`, etc.

3. **"header-max-length exceeded"**
   - Keep the entire first line under 100 characters

4. **Git hook not working**
   - Reinstall hooks: `npm run prepare`
   - Check `.husky/commit-msg` file exists

### Security Notice

**⚠️ IMPORTANT: Hook bypass is DISABLED for security reasons.**

This project enforces mandatory code quality checks that **cannot be bypassed** using `--no-verify` or similar methods. This ensures:

- All Go code is properly formatted
- All code passes linting and static analysis
- Dependencies are properly managed
- Commit messages follow conventions

**For emergencies:**
1. Fix formatting/linting issues first: `make fmt && make lint`
2. Contact a repository maintainer if immediate bypass is needed
3. Use secure commit methods: `make commit MESSAGE="your message"`

See `docs/hook-enforcement.md` for detailed security information.

### Check Hook Status
```bash
# List installed hooks
ls -la .husky/

# Test commitlint directly
echo "feat: test message" | npx commitlint
```

## Configuration

### commitlint.config.js
The project's commitlint configuration can be found in `commitlint.config.js`. Key settings:

- **Type enforcement**: Only allows predefined types
- **Scope suggestions**: Project-specific scopes
- **Length limits**: Header max 100 chars, body/footer max 72 chars
- **Case sensitivity**: Lowercase types and scopes

### Customization
To modify commit rules, edit `commitlint.config.js`:

```javascript
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Add custom rules here
    'type-enum': [2, 'always', ['feat', 'fix', /* ... */]],
    'scope-enum': [1, 'always', ['api', 'db', /* ... */]]
  }
};
```

## Resources

- [Conventional Commits Specification](https://conventionalcommits.org/)
- [Commitlint Documentation](https://commitlint.js.org/)
- [Husky Git Hooks](https://typicode.github.io/husky/)
- [Angular Commit Message Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#-commit-message-format)