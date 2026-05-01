#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawnSync } = require("node:child_process");

const repoRoot = path.resolve(__dirname, "..");

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || repoRoot,
    env: {
      ...process.env,
      ...options.env
    },
    encoding: "utf8"
  });
  const detail = [result.stdout, result.stderr, result.error && result.error.message]
    .filter(Boolean)
    .join("\n")
    .trim();

  if (options.expectStatus !== undefined) {
    assert.equal(result.status, options.expectStatus, detail);
  } else if (result.status !== 0) {
    assert.fail(detail === "" ? `${command} exited with status ${result.status}` : detail);
  }

  return result;
}

async function withTempDir(prefix, fn) {
  const tempDir = await fsp.mkdtemp(path.join(os.tmpdir(), prefix));
  try {
    return await fn(tempDir);
  } finally {
    await fsp.rm(tempDir, { recursive: true, force: true });
  }
}

async function writeFakeDate(binDir, value) {
  await fsp.mkdir(binDir, { recursive: true });
  const scriptPath = path.join(binDir, "date");
  await fsp.writeFile(scriptPath, `#!/usr/bin/env sh\nprintf '%s\\n' ${JSON.stringify(value)}\n`);
  await fsp.chmod(scriptPath, 0o755);
}

test("next release tag starts the monthly micro sequence at zero", async () => {
  await withTempDir("gig-release-tag-", async (tempDir) => {
    const repoDir = path.join(tempDir, "repo");
    const binDir = path.join(tempDir, "bin");
    await fsp.mkdir(repoDir, { recursive: true });
    await writeFakeDate(binDir, "2026.05");

    run("git", ["init", "-q"], { cwd: repoDir });
    await fsp.writeFile(path.join(repoDir, "README.md"), "release fixture\n");
    run("git", ["add", "README.md"], { cwd: repoDir });
    run("git", ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "initial"], {
      cwd: repoDir
    });

    const result = run("bash", [path.join(repoRoot, "scripts", "next-release-tag.sh")], {
      cwd: repoDir,
      env: {
        PATH: `${binDir}${path.delimiter}${process.env.PATH || ""}`
      }
    });

    assert.equal(result.stdout.trim(), "v2026.5.0");
  });
});

test("next release tag increments the monthly micro sequence", async () => {
  await withTempDir("gig-release-tag-", async (tempDir) => {
    const repoDir = path.join(tempDir, "repo");
    const binDir = path.join(tempDir, "bin");
    await fsp.mkdir(repoDir, { recursive: true });
    await writeFakeDate(binDir, "2026.05");

    run("git", ["init", "-q"], { cwd: repoDir });
    await fsp.writeFile(path.join(repoDir, "README.md"), "release fixture\n");
    run("git", ["add", "README.md"], { cwd: repoDir });
    run("git", ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "initial"], {
      cwd: repoDir
    });
    run("git", ["tag", "v2026.5.0"], { cwd: repoDir });
    run("git", ["tag", "v2026.5.1"], { cwd: repoDir });

    await fsp.writeFile(path.join(repoDir, "release.txt"), "second release\n");
    run("git", ["add", "release.txt"], { cwd: repoDir });
    run("git", ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "second"], {
      cwd: repoDir
    });

    const result = run("bash", [path.join(repoRoot, "scripts", "next-release-tag.sh")], {
      cwd: repoDir,
      env: {
        PATH: `${binDir}${path.delimiter}${process.env.PATH || ""}`
      }
    });

    assert.equal(result.stdout.trim(), "v2026.5.2");
  });
});

test("next release tag accounts for legacy date tags in the same month", async () => {
  await withTempDir("gig-release-tag-", async (tempDir) => {
    const repoDir = path.join(tempDir, "repo");
    const binDir = path.join(tempDir, "bin");
    await fsp.mkdir(repoDir, { recursive: true });
    await writeFakeDate(binDir, "2026.05");

    run("git", ["init", "-q"], { cwd: repoDir });
    await fsp.writeFile(path.join(repoDir, "README.md"), "release fixture\n");
    run("git", ["add", "README.md"], { cwd: repoDir });
    run("git", ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "initial"], {
      cwd: repoDir
    });
    run("git", ["tag", "v2026.5.0"], { cwd: repoDir });
    run("git", ["tag", "v2026.05.03"], { cwd: repoDir });

    await fsp.writeFile(path.join(repoDir, "release.txt"), "after legacy release\n");
    run("git", ["add", "release.txt"], { cwd: repoDir });
    run("git", ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "after legacy"], {
      cwd: repoDir
    });

    const result = run("bash", [path.join(repoRoot, "scripts", "next-release-tag.sh")], {
      cwd: repoDir,
      env: {
        PATH: `${binDir}${path.delimiter}${process.env.PATH || ""}`
      }
    });

    assert.equal(result.stdout.trim(), "v2026.5.4");
  });
});

test("prepare npm package keeps the canonical CalVer version", async () => {
  await withTempDir("gig-npm-version-", async (tempDir) => {
    const outputDir = path.join(tempDir, "package");

    run("node", ["scripts/prepare-npm-package.cjs", "v2026.5.2", outputDir]);

    const packageJson = JSON.parse(await fsp.readFile(path.join(outputDir, "package.json"), "utf8"));
    assert.equal(packageJson.version, "2026.5.2");
  });
});
