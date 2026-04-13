#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const packageJson = require(path.join(__dirname, "..", "..", "package.json"));
const binaryName = process.platform === "win32" ? "gig.exe" : "gig";
const binaryPath = path.join(__dirname, "..", "vendor", binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error(
    "gig is not installed correctly. Reinstall with `npm install -g " +
      packageJson.name +
      "` or run `npm rebuild -g " +
      packageJson.name +
      "`."
  );
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  env: {
    ...process.env,
    GIG_INSTALL_MODE: "npm",
    GIG_NPM_PACKAGE_NAME: packageJson.name,
    GIG_NPM_PACKAGE_VERSION: packageJson.version
  }
});

if (result.error) {
  console.error(`Failed to launch gig: ${result.error.message}`);
  process.exit(1);
}

process.exit(typeof result.status === "number" ? result.status : 1);
