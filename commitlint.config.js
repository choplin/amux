module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',     // New feature
        'fix',      // Bug fix
        'docs',     // Documentation only changes
        'style',    // Code style changes (formatting, etc)
        'refactor', // Code refactoring
        'test',     // Adding or updating tests
        'chore',    // Maintenance tasks
        'build',    // Build system changes
        'ci',       // CI configuration changes
        'perf',     // Performance improvements
        'revert'    // Revert a previous commit
      ]
    ],
    'subject-max-length': [2, 'always', 100],
    'subject-case': [2, 'never', ['upper-case', 'start-case']],
    'body-max-line-length': [2, 'always', 100]
  }
};