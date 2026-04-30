#!/usr/bin/env node
"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { Readable } = require("node:stream");
const { pipeline } = require("node:stream/promises");
const { spawnSync } = require("node:child_process");

function resolvePackageRoot(scriptDir) {
  if (fs.existsSync(path.join(scriptDir, "package.json"))) {
    return scriptDir;
  }
  return path.join(scriptDir, "..");
}

function resolveVendorDir(scriptDir) {
  return path.join(scriptDir, "vendor");
}

const packageRoot = resolvePackageRoot(__dirname);
const packageJson = require(path.join(packageRoot, "package.json"));

function resolveGitHubRepo(repository) {
  const repositoryURL =
    typeof repository === "string" ? repository : repository && repository.url;
  if (typeof repositoryURL !== "string" || repositoryURL.trim() === "") {
    return "phamhungptithcm/gig";
  }

  const normalized = repositoryURL
    .trim()
    .replace(/^git\+/, "")
    .replace(/\.git$/, "");
  const match = normalized.match(/github\.com[/:]([^/]+\/[^/]+)$/i);
  if (!match) {
    return "phamhungptithcm/gig";
  }

  return match[1];
}

function resolveReleaseBaseURL(repository, releaseTag) {
  const override = String(process.env.GIG_RELEASE_BASE_URL || "").trim();
  if (override !== "") {
    return override.replace(/\/+$/, "");
  }

  const repo = resolveGitHubRepo(repository);
  return `https://github.com/${repo}/releases/download/${releaseTag}`;
}

function normalizeReleaseTag(input) {
  const normalized = String(input || "").trim();
  if (normalized === "") {
    throw new Error("GIG_RELEASE_TAG must not be empty.");
  }
  return normalized.startsWith("v") ? normalized : `v${normalized}`;
}

function releaseTagFromPackageVersion(version) {
  if (process.env.GIG_RELEASE_TAG) {
    return normalizeReleaseTag(process.env.GIG_RELEASE_TAG);
  }

  const match = String(version || "").trim().match(/^(\d+)\.(\d+)\.(\d+)$/);
  if (!match) {
    throw new Error(
      `Package version ${JSON.stringify(
        version
      )} is not publishable. Use a released npm version or set GIG_RELEASE_TAG for local testing.`
    );
  }

  const [, year, month, day] = match;
  return `v${year}.${month.padStart(2, "0")}.${day.padStart(2, "0")}`;
}

function resolvePlatform() {
  switch (process.platform) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`Unsupported operating system: ${process.platform}`);
  }
}

function resolveArch() {
  switch (process.arch) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`Unsupported CPU architecture: ${process.arch}`);
  }
}

async function fetchOrThrow(url) {
  const response = await fetch(url, {
    headers: {
      "user-agent": `${packageJson.name}/${packageJson.version}`
    }
  });
  if (!response.ok) {
    throw new Error(`Request failed for ${url}: ${response.status} ${response.statusText}`);
  }
  return response;
}

async function downloadFile(url, destinationPath) {
  const response = await fetchOrThrow(url);
  await pipeline(Readable.fromWeb(response.body), fs.createWriteStream(destinationPath));
}

async function fetchText(url) {
  const response = await fetchOrThrow(url);
  return response.text();
}

function checksumForFile(filePath) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash("sha256");
    const input = fs.createReadStream(filePath);

    input.on("error", reject);
    input.on("data", (chunk) => hash.update(chunk));
    input.on("end", () => resolve(hash.digest("hex")));
  });
}

function expectedChecksum(checksumText, assetName) {
  for (const line of checksumText.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (trimmed === "") {
      continue;
    }

    const match = trimmed.match(/^([a-fA-F0-9]{64})\s+(.+)$/);
    if (match && match[2] === assetName) {
      return match[1].toLowerCase();
    }
  }

  return "";
}

function runCommand(command, args) {
  const result = spawnSync(command, args, {
    stdio: "pipe",
    encoding: "utf8"
  });

  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    const detail = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
    throw new Error(detail === "" ? `${command} exited with status ${result.status}` : detail);
  }
}

function extractArchive(archivePath, extractDir, platform) {
  if (platform === "windows") {
    runCommand("powershell.exe", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      `Expand-Archive -LiteralPath '${archivePath.replace(/'/g, "''")}' -DestinationPath '${extractDir.replace(/'/g, "''")}' -Force`
    ]);
    return;
  }

  runCommand("tar", ["-xzf", archivePath, "-C", extractDir]);
}

async function install() {
  const releaseTag = releaseTagFromPackageVersion(packageJson.version);
  const versionWithoutV = releaseTag.replace(/^v/, "");
  const platform = resolvePlatform();
  const arch = resolveArch();
  const extension = platform === "windows" ? "zip" : "tar.gz";
  const binaryName = platform === "windows" ? "gig.exe" : "gig";
  const assetName = `gig_${versionWithoutV}_${platform}_${arch}.${extension}`;
  const checksumsName = `gig_${versionWithoutV}_checksums.txt`;
  const baseURL = resolveReleaseBaseURL(packageJson.repository, releaseTag);
  const vendorDir = resolveVendorDir(__dirname);
  const tmpDir = await fsp.mkdtemp(path.join(os.tmpdir(), "gig-npm-"));

  try {
    const archivePath = path.join(tmpDir, assetName);
    const extractDir = path.join(tmpDir, "extract");

    await fsp.mkdir(extractDir, { recursive: true });

    const [checksums, _] = await Promise.all([
      fetchText(`${baseURL}/${checksumsName}`),
      downloadFile(`${baseURL}/${assetName}`, archivePath)
    ]);

    const expected = expectedChecksum(checksums, assetName);
    if (expected === "") {
      throw new Error(`Could not find a checksum for ${assetName} in ${checksumsName}.`);
    }

    const actual = await checksumForFile(archivePath);
    if (actual !== expected) {
      throw new Error(
        `Checksum verification failed for ${assetName}. Expected ${expected} but got ${actual}.`
      );
    }

    extractArchive(archivePath, extractDir, platform);

    const sourceBinary = path.join(extractDir, binaryName);
    await fsp.access(sourceBinary);

    await fsp.rm(vendorDir, { recursive: true, force: true });
    await fsp.mkdir(vendorDir, { recursive: true });

    const targetBinary = path.join(vendorDir, binaryName);
    await fsp.copyFile(sourceBinary, targetBinary);
    if (platform !== "windows") {
      await fsp.chmod(targetBinary, 0o755);
    }

    await fsp.writeFile(
      path.join(vendorDir, "INSTALL_RECEIPT.json"),
      JSON.stringify(
        {
          packageName: packageJson.name,
          packageVersion: packageJson.version,
          releaseTag,
          assetName,
          installedAt: new Date().toISOString()
        },
        null,
        2
      ) + "\n"
    );

    console.log(`gig ${packageJson.version} installed for ${platform}/${arch}`);
  } finally {
    await fsp.rm(tmpDir, { recursive: true, force: true });
  }
}

if (require.main === module) {
  install().catch((error) => {
    console.error(`gig postinstall failed: ${error.message}`);
    process.exit(1);
  });
}

module.exports = {
  resolvePackageRoot,
  resolveReleaseBaseURL,
  resolveVendorDir
};
