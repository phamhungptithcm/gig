#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const crypto = require("node:crypto");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawn, spawnSync } = require("node:child_process");

const repoRoot = path.resolve(__dirname, "..");
const releaseTag = "v2099.1.0";
const isWindows = process.platform === "win32";
const binaryName = isWindows ? "gig.exe" : "gig";
const archiveExtension = isWindows ? "zip" : "tar.gz";
const platformName = isWindows ? "windows" : process.platform;
const archName = process.arch === "x64" ? "amd64" : process.arch;
const stableAssetName = `gig_${platformName}_${archName}.${archiveExtension}`;
const versionedAssetName = `gig_${releaseTag.slice(1)}_${platformName}_${archName}.${archiveExtension}`;
const checksumName = `gig_${releaseTag.slice(1)}_checksums.txt`;

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd || repoRoot,
    env: {
      ...process.env,
      ...options.env
    },
    encoding: "utf8",
    shell: options.shell || false,
    stdio: options.stdio || "pipe"
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

async function createReleaseFixture(rootDir) {
  const buildDir = path.join(rootDir, "build");
  const distDir = path.join(rootDir, "dist");
  await fsp.mkdir(buildDir, { recursive: true });
  await fsp.mkdir(distDir, { recursive: true });

  const binaryPath = path.join(buildDir, binaryName);
  run("go", ["build", "-trimpath", "-o", binaryPath, "./cmd/gig"]);

  const archivePath = path.join(distDir, stableAssetName);
  if (isWindows) {
    run("powershell.exe", [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      `Compress-Archive -LiteralPath '${binaryPath.replace(/'/g, "''")}' -DestinationPath '${archivePath.replace(/'/g, "''")}' -Force`
    ]);
  } else {
    run("tar", ["-czf", archivePath, "-C", buildDir, binaryName]);
  }
  await fsp.copyFile(archivePath, path.join(distDir, versionedAssetName));

  const stableHash = crypto.createHash("sha256");
  stableHash.update(await fsp.readFile(archivePath));
  const checksumText = [
    `${stableHash.digest("hex")}  ${stableAssetName}`,
    `${crypto.createHash("sha256").update(await fsp.readFile(path.join(distDir, versionedAssetName))).digest("hex")}  ${versionedAssetName}`,
    ""
  ].join("\n");
  await fsp.writeFile(path.join(distDir, checksumName), checksumText);

  return distDir;
}

async function startStaticServer(rootDir) {
  const serverScript = `
const fsp = require("node:fs/promises");
const http = require("node:http");
const path = require("node:path");

const rootDir = process.argv[1];
const host = process.argv[2];

const server = http.createServer(async (req, res) => {
    try {
      const pathname = decodeURIComponent(new URL(req.url, \`http://\${req.headers.host}\`).pathname);
      const relativePath = pathname.replace(/^\\/+/, "");
      const filePath = path.resolve(rootDir, relativePath);
      const realRoot = path.resolve(rootDir);
      if (!filePath.startsWith(realRoot + path.sep) && filePath !== realRoot) {
        res.writeHead(403);
        res.end("forbidden");
        return;
      }
      const data = await fsp.readFile(filePath);
      res.writeHead(200, {
        "content-length": String(data.length),
        "content-type": relativePath.endsWith(".txt") ? "text/plain; charset=utf-8" : "application/octet-stream"
      });
      res.end(data);
    } catch (error) {
      res.writeHead(error && error.code === "ENOENT" ? 404 : 500);
      res.end("not found");
    }
  });

server.listen(0, host, () => {
  const address = server.address();
  process.stdout.write(String(address.port) + "\\n");
});
`;

  return await new Promise((resolve, reject) => {
    const child = spawn(process.execPath, ["-e", serverScript, rootDir, "127.0.0.1"], {
      cwd: repoRoot,
      stdio: ["ignore", "pipe", "pipe"]
    });

    let stdout = "";
    let stderr = "";
    child.once("error", reject);
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
      const newlineIndex = stdout.indexOf("\n");
      if (newlineIndex === -1) {
        return;
      }
      const port = Number(stdout.slice(0, newlineIndex).trim());
      if (!Number.isFinite(port) || port <= 0) {
        reject(new Error(`static asset server reported an invalid port: ${stdout}`));
        return;
      }
      resolve({
        child,
        url: `http://127.0.0.1:${port}`
      });
    });
    child.once("exit", () => {
      if (stdout.indexOf("\n") === -1) {
        reject(new Error(stderr.trim() === "" ? "static asset server exited before reporting a port" : stderr.trim()));
      }
    });
  });
}

test("direct installer can install from a local release fixture", async () => {
  const fixtureRoot = await fsp.mkdtemp(path.join(os.tmpdir(), "gig-direct-install-"));
  const installDir = path.join(fixtureRoot, "bin");
  const profilePath = path.join(fixtureRoot, "profile");
  const distDir = await createReleaseFixture(fixtureRoot);
  const serverInfo = await startStaticServer(distDir);

  try {
    const env = {
      GIG_RELEASE_BASE_URL: serverInfo.url,
      GIG_SHELL_PROFILE: profilePath
    };

    if (isWindows) {
      env.GIG_SKIP_PATH_UPDATE = "1";
      run("powershell.exe", [
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        path.join(repoRoot, "scripts", "install.ps1"),
        "-Version",
        releaseTag,
        "-Repo",
        "acme/gig",
        "-InstallDir",
        installDir
      ], { env });
    } else {
      run("sh", [
        path.join(repoRoot, "scripts", "install.sh"),
        "--version",
        releaseTag,
        "--repo",
        "acme/gig",
        "--install-dir",
        installDir
      ], { env });
    }

    const installedBinary = path.join(installDir, binaryName);
    assert.equal(fs.existsSync(installedBinary), true);
    const version = run(installedBinary, ["version"]);
    assert.match(version.stdout, /^gig /m);

    if (!isWindows) {
      assert.match(await fsp.readFile(profilePath, "utf8"), new RegExp(installDir.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
    }
  } finally {
    if (serverInfo.child && !serverInfo.child.killed) {
      serverInfo.child.kill();
      await new Promise((resolve) => serverInfo.child.once("exit", resolve));
    }
    await fsp.rm(fixtureRoot, { recursive: true, force: true });
  }
});
