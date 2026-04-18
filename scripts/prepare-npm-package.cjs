#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const fsp = require("node:fs/promises");
const path = require("node:path");

function usage() {
  console.error("usage: prepare-npm-package.cjs <release-tag> <output-dir>");
  process.exit(1);
}

function normalizeReleaseTag(input) {
  const value = String(input || "").trim();
  if (!/^v\d{4}\.\d{2}\.\d{2}$/.test(value)) {
    throw new Error(`release tag ${JSON.stringify(value)} must use vYYYY.MM.DD`);
  }
  return value;
}

function releaseTagToPackageVersion(tag) {
  const [, year, month, day] = tag.match(/^v(\d{4})\.(\d{2})\.(\d{2})$/);
  return `${year}.${Number(month)}.${Number(day)}`;
}

async function copyDirectory(sourceDir, targetDir) {
  await fsp.mkdir(targetDir, { recursive: true });
  const entries = await fsp.readdir(sourceDir, { withFileTypes: true });

  for (const entry of entries) {
    const sourcePath = path.join(sourceDir, entry.name);
    const targetPath = path.join(targetDir, entry.name);
    if (entry.isDirectory()) {
      await copyDirectory(sourcePath, targetPath);
      continue;
    }
    if (entry.isFile()) {
      await fsp.copyFile(sourcePath, targetPath);
    }
  }
}

async function main() {
  if (process.argv.length !== 4) {
    usage();
  }

  const releaseTag = normalizeReleaseTag(process.argv[2]);
  const outputDir = path.resolve(process.argv[3]);
  const repoRoot = path.resolve(__dirname, "..");
  const packageTemplatePath = path.join(repoRoot, "package.json");
  const packageTemplate = JSON.parse(await fsp.readFile(packageTemplatePath, "utf8"));
  const packageVersion = releaseTagToPackageVersion(releaseTag);

  await fsp.rm(outputDir, { recursive: true, force: true });
  await fsp.mkdir(outputDir, { recursive: true });

  await copyDirectory(path.join(repoRoot, "npm", "bin"), path.join(outputDir, "bin"));
  await fsp.copyFile(path.join(repoRoot, "npm", "install.cjs"), path.join(outputDir, "install.cjs"));
  await fsp.copyFile(path.join(repoRoot, "README.md"), path.join(outputDir, "README.md"));

  const publishPackage = {
    ...packageTemplate,
    version: packageVersion,
    bin: {
      gig: "./bin/gig.cjs"
    },
    files: [
      "bin",
      "install.cjs",
      "README.md"
    ],
    scripts: {
      ...packageTemplate.scripts,
      postinstall: "node ./install.cjs"
    }
  };

  await fsp.writeFile(
    path.join(outputDir, "package.json"),
    JSON.stringify(publishPackage, null, 2) + "\n"
  );

  if (!fs.existsSync(path.join(outputDir, "bin", "gig.cjs"))) {
    throw new Error("npm wrapper was not copied into the package staging directory");
  }

  console.log(`Prepared npm package ${publishPackage.name}@${publishPackage.version} in ${outputDir}`);
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
