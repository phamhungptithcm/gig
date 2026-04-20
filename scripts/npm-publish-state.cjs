#!/usr/bin/env node

const fs = require('node:fs');

function parseJSONInput(input) {
  const trimmed = input.trim();
  if (trimmed === '') {
    return null;
  }
  return JSON.parse(trimmed);
}

function trustEntries(body) {
  if (!body) {
    return [];
  }
  return Array.isArray(body) ? body : [body];
}

function hasMatchingTrustedPublisher(body, expected) {
  return trustEntries(body).some((entry) => {
    const claims = entry?.claims || {};
    const workflowFile = claims.workflow_ref?.file || '';
    const environment = claims.environment || '';
    return (
      entry?.type === 'github' &&
      claims.repository === expected.repository &&
      workflowFile === expected.workflowFile &&
      environment === expected.environment
    );
  });
}

function collaboratorAccess(body, user) {
  if (!body || typeof body !== 'object') {
    return '';
  }

  const access = body[user];
  switch (access) {
    case 'write':
    case 'read-write':
      return 'read-write';
    case 'read':
    case 'read-only':
      return 'read-only';
    default:
      return '';
  }
}

function main() {
  const command = process.argv[2];
  const input = fs.readFileSync(0, 'utf8');
  const body = parseJSONInput(input);

  switch (command) {
    case 'trust-match': {
      const repository = process.argv[3] || '';
      const workflowFile = process.argv[4] || '';
      const environment = process.argv[5] || '';
      const matched = hasMatchingTrustedPublisher(body, {
        repository,
        workflowFile,
        environment,
      });
      process.exit(matched ? 0 : 1);
      break;
    }
    case 'collaborator-access': {
      const user = process.argv[3] || '';
      const access = collaboratorAccess(body, user);
      if (access !== '') {
        process.stdout.write(`${access}\n`);
        process.exit(0);
      }
      process.exit(1);
      break;
    }
    default:
      console.error('usage: npm-publish-state.cjs <trust-match|collaborator-access> [...]');
      process.exit(2);
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  collaboratorAccess,
  hasMatchingTrustedPublisher,
};
