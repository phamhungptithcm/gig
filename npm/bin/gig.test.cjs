#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { resolvePackageRoot } = require("./gig.cjs");

test("resolve wrapper package root for source layout", async () => {
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "gig-wrapper-source-"));
  const repoRoot = path.join(tempRoot, "repo");
  const scriptDir = path.join(repoRoot, "npm", "bin");

  await fs.promises.mkdir(scriptDir, { recursive: true });
  await fs.promises.writeFile(path.join(repoRoot, "package.json"), "{}\n");

  assert.equal(resolvePackageRoot(scriptDir), repoRoot);

  await fs.promises.rm(tempRoot, { recursive: true, force: true });
});

test("resolve wrapper package root for published layout", async () => {
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "gig-wrapper-published-"));
  const packageRoot = path.join(tempRoot, "@hunpeolabs", "gig");
  const scriptDir = path.join(packageRoot, "bin");

  await fs.promises.mkdir(scriptDir, { recursive: true });
  await fs.promises.writeFile(path.join(packageRoot, "package.json"), "{}\n");

  assert.equal(resolvePackageRoot(scriptDir), packageRoot);

  await fs.promises.rm(tempRoot, { recursive: true, force: true });
});
