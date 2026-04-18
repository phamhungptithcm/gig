const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'gig-npm-publish-'));
}

function writeExecutable(filePath, contents) {
  fs.writeFileSync(filePath, contents);
  fs.chmodSync(filePath, 0o755);
}

function runScript(scriptName, args, env = {}) {
  const tempDir = makeTempDir();
  const binDir = path.join(tempDir, 'bin');
  fs.mkdirSync(binDir, { recursive: true });

  const packageDir = path.join(tempDir, 'package');
  fs.mkdirSync(packageDir, { recursive: true });
  fs.writeFileSync(
    path.join(packageDir, 'package.json'),
    JSON.stringify(
      {
        name: '@hunpeolabs/gig',
        version: '2026.4.18',
        publishConfig: {
          access: 'public',
          registry: 'https://registry.npmjs.org/',
        },
      },
      null,
      2
    )
  );

  const statePath = path.join(tempDir, 'npm-state.json');
  fs.writeFileSync(statePath, JSON.stringify(env.npmState || {}, null, 2));

  writeExecutable(
    path.join(binDir, 'npm'),
    `#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const state = JSON.parse(fs.readFileSync(${JSON.stringify(statePath)}, 'utf8'));
const args = process.argv.slice(2);
const packageName = state.packageName || '@hunpeolabs/gig';
const registryUrl = state.registryUrl || 'https://registry.npmjs.org/';
function logStderr(message) {
  process.stderr.write(String(message) + '\\n');
}
function emitJSON(value) {
  process.stdout.write(JSON.stringify(value));
  if (process.stdout.isTTY) {
    process.stdout.write('\\n');
  }
}
if (args[0] === 'whoami') {
  if (state.whoamiError) {
    logStderr(state.whoamiError);
    process.exit(1);
  }
  process.stdout.write((state.npmUser || 'hunpeo97') + '\\n');
  process.exit(0);
}
if (args[0] === 'view') {
  const spec = args[1];
  const target = spec.split('@').length > 2 ? '@' + spec.split('@').slice(1).join('@') : spec;
  if (target === packageName || spec.startsWith(packageName + '@')) {
    if (state.packageExists === false) {
      logStderr('E404');
      process.exit(1);
    }
    process.stdout.write((state.packageVersion || '2026.4.14') + '\\n');
    process.exit(0);
  }
}
if (args[0] === 'trust' && args[1] === 'list') {
  if (state.trustListError) {
    logStderr(state.trustListError);
    process.exit(1);
  }
  emitJSON(state.trustList || []);
  process.exit(0);
}
if (args[0] === 'access' && args[1] === 'list' && args[2] === 'collaborators') {
  if (state.collaboratorsError) {
    logStderr(state.collaboratorsError);
    process.exit(1);
  }
  emitJSON(state.collaborators || {});
  process.exit(0);
}
if (args[0] === 'access' && args[1] === 'list' && args[2] === 'packages') {
  if (state.scopePackagesError) {
    logStderr(state.scopePackagesError);
    process.exit(1);
  }
  emitJSON(state.scopePackages || {});
  process.exit(0);
}
if (args[0] === 'pack') {
  const tarballName = state.tarballName || 'hunpeolabs-gig-2026.4.18.tgz';
  const cwd = process.cwd();
  fs.writeFileSync(path.join(cwd, tarballName), 'fake tarball');
  process.stdout.write(tarballName + '\\n');
  process.exit(0);
}
if (args[0] === 'publish') {
  if (state.publishError) {
    logStderr(state.publishError);
    process.exit(1);
  }
  process.stdout.write('published\\n');
  process.exit(0);
}
logStderr('Unexpected npm invocation: ' + args.join(' '));
process.exit(1);
`
  );

  const scriptPath = path.join(process.cwd(), 'scripts', scriptName);
  const result = spawnSync(scriptPath, args, {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PATH: `${binDir}:${process.env.PATH}`,
      ...env.extraEnv,
    },
    encoding: 'utf8',
  });

  fs.rmSync(tempDir, { recursive: true, force: true });
  return { result, packageDir };
}

test('resolve-npm-publish-mode prefers trusted publishing for an existing package', () => {
  const { result } = runScript('resolve-npm-publish-mode.sh', ['@hunpeolabs/gig'], {
    npmState: {
      packageExists: true,
      trustList: [
        {
          type: 'github',
          claims: {
            repository: 'phamhungptithcm/gig',
            workflow_ref: { file: 'release.yml' },
            environment: 'npm-release',
          },
        },
      ],
    },
    extraEnv: {
      NPM_PUBLISH_TOKEN: 'stale-token',
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /mode=trusted/);
  assert.match(result.stdout, /trusted_publisher_configured=true/);
});

test('resolve-npm-publish-mode falls back to token when trusted publishing is not configured', () => {
  const { result } = runScript('resolve-npm-publish-mode.sh', ['@hunpeolabs/gig'], {
    npmState: {
      packageExists: true,
      trustList: [],
    },
    extraEnv: {
      NPM_PUBLISH_TOKEN: 'fallback-token',
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /mode=token/);
  assert.match(result.stdout, /trusted_publisher_configured=false/);
});

test('verify-npm-publish-auth accepts an existing package when the token user has write access', () => {
  const { result } = runScript('verify-npm-publish-auth.sh', ['@hunpeolabs/gig'], {
    npmState: {
      packageExists: true,
      npmUser: 'hunpeo97',
      collaborators: {
        hunpeo97: 'write',
      },
    },
    extraEnv: {
      NODE_AUTH_TOKEN: 'token',
    },
  });

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /package_access=read-write/);
});

test('verify-npm-publish-auth fails clearly when the token user cannot publish an existing package', () => {
  const { result } = runScript('verify-npm-publish-auth.sh', ['@hunpeolabs/gig'], {
    npmState: {
      packageExists: true,
      npmUser: 'automation-user',
      collaborators: {},
    },
    extraEnv: {
      NODE_AUTH_TOKEN: 'token',
    },
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /does not have read-write publish access/);
  assert.match(result.stderr, /prefer npm trusted publishing/);
});

test('publish-npm-package adds a targeted message for existing-package permission 404s', () => {
  const tempDir = makeTempDir();
  const packageDir = path.join(tempDir, 'package');
  fs.mkdirSync(packageDir, { recursive: true });
  fs.writeFileSync(
    path.join(packageDir, 'package.json'),
    JSON.stringify(
      {
        name: '@hunpeolabs/gig',
        version: '2026.4.18',
      },
      null,
      2
    )
  );

  const binDir = path.join(tempDir, 'bin');
  fs.mkdirSync(binDir, { recursive: true });
  const statePath = path.join(tempDir, 'npm-state.json');
  fs.writeFileSync(
    statePath,
    JSON.stringify(
      {
        packageExists: true,
        publishError:
          'npm error code E404\\nnpm error 404 Not Found - PUT https://registry.npmjs.org/@hunpeolabs%2fgig - Not found\\nnpm error 404 The requested resource could not be found or you do not have permission to access it.',
      },
      null,
      2
    )
  );
  writeExecutable(
    path.join(binDir, 'npm'),
    `#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const state = JSON.parse(fs.readFileSync(${JSON.stringify(statePath)}, 'utf8'));
const args = process.argv.slice(2);
if (args[0] === 'pack') {
  const tarballName = 'hunpeolabs-gig-2026.4.18.tgz';
  fs.writeFileSync(path.join(process.cwd(), tarballName), 'fake tarball');
  process.stdout.write(tarballName + '\\n');
  process.exit(0);
}
if (args[0] === 'view') {
  if (state.packageExists === false) {
    process.stderr.write('E404\\n');
    process.exit(1);
  }
  process.stdout.write('2026.4.14\\n');
  process.exit(0);
}
if (args[0] === 'publish') {
  process.stderr.write(state.publishError + '\\n');
  process.exit(1);
}
process.stderr.write('Unexpected npm invocation: ' + args.join(' ') + '\\n');
process.exit(1);
`
  );

  const result = spawnSync(path.join(process.cwd(), 'scripts', 'publish-npm-package.sh'), [packageDir], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PATH: `${binDir}:${process.env.PATH}`,
    },
    encoding: 'utf8',
  });

  fs.rmSync(tempDir, { recursive: true, force: true });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /already exists on npm, but the authenticated identity cannot publish it/);
  assert.match(result.stderr, /replace NPM_PUBLISH_TOKEN with a package owner or collaborator token/);
});
