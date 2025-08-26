module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Type must be one of the following
    'type-enum': [2, 'always', [
      'feat',     // A new feature
      'fix',      // A bug fix
      'docs',     // Documentation only changes
      'style',    // Changes that do not affect the meaning of the code
      'refactor', // A code change that neither fixes a bug nor adds a feature
      'perf',     // A code change that improves performance
      'test',     // Adding missing tests or correcting existing tests
      'build',    // Changes that affect the build system or external dependencies
      'ci',       // Changes to our CI configuration files and scripts
      'chore',    // Other changes that don't modify src or test files
      'revert',   // Reverts a previous commit
    ]],
    
    // Subject must not be empty
    'subject-empty': [2, 'never'],
    
    // Subject must not end with a period
    'subject-full-stop': [2, 'never', '.'],
    
    // Subject must be lowercase
    'subject-case': [2, 'always', 'lower-case'],
    
    // Header (type + scope + subject) max length
    'header-max-length': [2, 'always', 100],
    
    // Body should wrap at 72 characters
    'body-max-line-length': [1, 'always', 72],
    
    // Footer should wrap at 72 characters
    'footer-max-line-length': [1, 'always', 72],
    
    // Scope should be lowercase
    'scope-case': [2, 'always', 'lower-case'],
    
    // Allow specific scopes for this project
    'scope-enum': [1, 'always', [
      // API related
      'api',
      'handlers',
      'middleware',
      'auth',
      'validation',
      
      // Database/Repository
      'db',
      'repository',
      'migrations',
      'collections',
      
      // Domain/Business Logic
      'domain',
      'services',
      'models',
      
      // Infrastructure
      'config',
      'container',
      'logging',
      'monitoring',
      
      // Testing
      'tests',
      'integration',
      'unit',
      'e2e',
      'mocks',
      
      // DevOps/CI
      'ci',
      'docker',
      'deployment',
      'scripts',
      
      // Documentation
      'docs',
      'readme',
      'changelog',
      
      // Dependencies
      'deps',
      'security',
      
      // Project specific
      'tasks',
      'projects',
      'users',
      'comments',
      'tags'
    ]]
  },
  
  // Custom parser options
  parserPreset: {
    parserOpts: {
      // Allow longer subject lines for detailed descriptions
      headerPattern: /^(\w*)(?:\(([\w\$\.\-\* ]*)\))?\: (.*)$/,
      headerCorrespondence: ['type', 'scope', 'subject'],
    }
  },
  
  // Plugin configuration
  plugins: [],
  
  // Help URL for developers
  helpUrl: 'https://github.com/conventional-changelog/commitlint/#what-is-commitlint'
};