#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { resolvePackageRoot, resolveReleaseBaseURL, resolveVendorDir } = require("./install.cjs");

test("resolve source-layout package root and vendor dir", async () => {
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "gig-install-source-"));
  const repoRoot = path.join(tempRoot, "repo");
  const scriptDir = path.join(repoRoot, "npm");

  await fs.promises.mkdir(scriptDir, { recursive: true });
  await fs.promises.writeFile(path.join(repoRoot, "package.json"), "{}\n");

  assert.equal(resolvePackageRoot(scriptDir), repoRoot);
  assert.equal(resolveVendorDir(scriptDir), path.join(repoRoot, "npm", "vendor"));

  await fs.promises.rm(tempRoot, { recursive: true, force: true });
});

test("resolve published-layout package root and vendor dir", async () => {
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "gig-install-published-"));
  const packageRoot = path.join(tempRoot, "@hunpeolabs", "gig");

  await fs.promises.mkdir(packageRoot, { recursive: true });
  await fs.promises.writeFile(path.join(packageRoot, "package.json"), "{}\n");

  assert.equal(resolvePackageRoot(packageRoot), packageRoot);
  assert.equal(resolveVendorDir(packageRoot), path.join(packageRoot, "vendor"));

  await fs.promises.rm(tempRoot, { recursive: true, force: true });
});

test("resolve release base URL from the GitHub repository", () => {
  delete process.env.GIG_RELEASE_BASE_URL;

  assert.equal(
    resolveReleaseBaseURL("git+https://github.com/phamhungptithcm/gig.git", "v2026.04.20"),
    "https://github.com/phamhungptithcm/gig/releases/download/v2026.04.20"
  );
});

test("resolve release base URL from environment override", () => {
  process.env.GIG_RELEASE_BASE_URL = "http://127.0.0.1:8787/releases/";

  try {
    assert.equal(
      resolveReleaseBaseURL("git+https://github.com/phamhungptithcm/gig.git", "v2026.04.20"),
      "http://127.0.0.1:8787/releases"
    );
  } finally {
    delete process.env.GIG_RELEASE_BASE_URL;
  }
});
