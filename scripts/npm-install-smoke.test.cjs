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
const releaseTag = "v2099.01.01";
const npmVersion = "2099.1.1";
const serverHost = "127.0.0.1";
const isWindows = process.platform === "win32";
const binaryName = isWindows ? "gig.exe" : "gig";
const archiveExtension = isWindows ? "zip" : "tar.gz";
const platformName = isWindows ? "windows" : process.platform;
const archName = process.arch === "x64" ? "amd64" : process.arch;
const assetName = `gig_${releaseTag.slice(1)}_${platformName}_${archName}.${archiveExtension}`;
const checksumName = `gig_${releaseTag.slice(1)}_checksums.txt`;
const smokeCommands = [
  { args: ["--help"], expect: /Remote-first release audit CLI/ },
  { args: ["inspect", "--help"], expect: /Show the full ticket story across repositories/ },
  { args: ["verify", "--help"], expect: /safe, warning, or blocked release verdict/ },
  { args: ["manifest", "--help"], expect: /Generate a release packet for QA, client, and release review/ },
  { args: ["assist", "--help"], expect: /gig assist audit/ },
  { args: ["resume", "--help"], expect: /saved assist session/ },
  { args: ["version"], expect: /^gig /m }
];

let fixtureRoot;
let packageDir;
let distDir;
let serverProcess;
let releaseBaseURL;

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

  if (options.expectStatus !== undefined) {
    assert.equal(
      result.status,
      options.expectStatus,
      [result.stdout, result.stderr].filter(Boolean).join("\n").trim()
    );
  } else if (result.status !== 0) {
    assert.fail([result.stdout, result.stderr].filter(Boolean).join("\n").trim());
  }

  return result;
}

async function createCurrentPlatformReleaseFixture(rootDir) {
  const buildDir = path.join(rootDir, "build");
  const distPath = path.join(rootDir, "dist");
  await fsp.mkdir(buildDir, { recursive: true });
  await fsp.mkdir(distPath, { recursive: true });

  const binaryPath = path.join(buildDir, binaryName);
  run("go", ["build", "-trimpath", "-o", binaryPath, "./cmd/gig"]);

  const archivePath = path.join(distPath, assetName);
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

  const hash = crypto.createHash("sha256");
  hash.update(await fsp.readFile(archivePath));
  await fsp.writeFile(path.join(distPath, checksumName), `${hash.digest("hex")}  ${assetName}\n`);
}

async function startStaticServer(rootDir) {
  const serverScript = `
const fs = require("node:fs/promises");
const http = require("node:http");
const path = require("node:path");

const rootDir = process.argv[1];
const host = process.argv[2];

const server = http.createServer(async (req, res) => {
  try {
    const pathname = decodeURIComponent(new URL(req.url, \`http://\${req.headers.host}\`).pathname);
    const relativePath = pathname.replace(/^\\/+/, "");
    const filePath = path.join(rootDir, relativePath);
    const realRoot = path.resolve(rootDir);
    const realPath = path.resolve(filePath);
    if (!realPath.startsWith(realRoot + path.sep) && realPath !== realRoot) {
      res.writeHead(403);
      res.end("forbidden");
      return;
    }

    const data = await fs.readFile(realPath);
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
    const child = spawn(process.execPath, ["-e", serverScript, rootDir, serverHost], {
      cwd: repoRoot,
      stdio: ["ignore", "pipe", "pipe"]
    });

    let stdout = "";
    let stderr = "";

    const onFailure = (error) => {
      reject(
        error ||
          new Error(stderr.trim() === "" ? "static asset server exited before reporting a port" : stderr.trim())
      );
    };

    child.once("error", onFailure);
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
        onFailure(new Error(`static asset server reported an invalid port: ${stdout}`));
        return;
      }

      resolve({
        child,
        url: `http://${serverHost}:${port}`
      });
    });
    child.once("exit", () => {
      if (stdout.indexOf("\n") === -1) {
        onFailure();
      }
    });
  });
}

function globalPrefixBinDir(prefixDir) {
  return isWindows ? prefixDir : path.join(prefixDir, "bin");
}

function globalPackageRoot(prefixDir) {
  if (isWindows) {
    return path.join(prefixDir, "node_modules", "@hunpeolabs", "gig");
  }
  return path.join(prefixDir, "lib", "node_modules", "@hunpeolabs", "gig");
}

function localPackageRoot(projectDir) {
  return path.join(projectDir, "node_modules", "@hunpeolabs", "gig");
}

function readInstallReceipt(packageRoot) {
  const receiptPath = path.join(packageRoot, "vendor", "INSTALL_RECEIPT.json");
  return JSON.parse(fs.readFileSync(receiptPath, "utf8"));
}

function combinedOutput(result) {
  return [result.stdout, result.stderr].filter(Boolean).join("\n");
}

function assertSmokeCommands(runCommand) {
  for (const command of smokeCommands) {
    const result = runCommand(command.args);
    assert.match(
      combinedOutput(result),
      command.expect,
      `expected ${command.args.join(" ")} to match ${String(command.expect)}`
    );
  }
}

test.before(async () => {
  fixtureRoot = await fsp.mkdtemp(path.join(os.tmpdir(), "gig-npm-smoke-"));
  distDir = path.join(fixtureRoot, "dist");
  packageDir = path.join(fixtureRoot, "npm-package");

  await createCurrentPlatformReleaseFixture(fixtureRoot);
  run("node", ["scripts/prepare-npm-package.cjs", releaseTag, packageDir]);

  const serverInfo = await startStaticServer(distDir);
  serverProcess = serverInfo.child;
  releaseBaseURL = serverInfo.url;
});

test.after(async () => {
  if (serverProcess && !serverProcess.killed) {
    serverProcess.kill();
    await new Promise((resolve) => serverProcess.once("exit", resolve));
  }
  if (fixtureRoot) {
    await fsp.rm(fixtureRoot, { recursive: true, force: true });
  }
});

test("local npm install can run gig through npx", async () => {
  const projectDir = path.join(fixtureRoot, "local-project");
  await fsp.mkdir(projectDir, { recursive: true });
  await fsp.writeFile(
    path.join(projectDir, "package.json"),
    JSON.stringify({ name: "gig-local-smoke", private: true }, null, 2) + "\n"
  );

  run(
    "npm",
    ["install", "--no-audit", "--no-fund", packageDir],
    {
      cwd: projectDir,
      env: {
        GIG_RELEASE_BASE_URL: releaseBaseURL
      }
    }
  );

  const packageRoot = localPackageRoot(projectDir);
  assert.equal(fs.existsSync(path.join(packageRoot, "bin", "gig.cjs")), true);
  assert.equal(fs.existsSync(path.join(packageRoot, "vendor", binaryName)), true);

  const receipt = readInstallReceipt(packageRoot);
  assert.equal(receipt.packageVersion, npmVersion);
  assert.equal(receipt.assetName, assetName);

  assertSmokeCommands((args) =>
    run("npx", ["--no-install", "gig", ...args], {
      cwd: projectDir
    })
  );
});

test("global npm install can run gig from the shim path", async () => {
  const prefixDir = path.join(fixtureRoot, "global-prefix");
  await fsp.mkdir(prefixDir, { recursive: true });

  run(
    "npm",
    ["install", "-g", "--no-audit", "--no-fund", "--prefix", prefixDir, packageDir],
    {
      env: {
        GIG_RELEASE_BASE_URL: releaseBaseURL
      }
    }
  );

  const packageRoot = globalPackageRoot(prefixDir);
  assert.equal(fs.existsSync(path.join(packageRoot, "bin", "gig.cjs")), true);
  assert.equal(fs.existsSync(path.join(packageRoot, "vendor", binaryName)), true);

  const receipt = readInstallReceipt(packageRoot);
  assert.equal(receipt.packageVersion, npmVersion);
  assert.equal(receipt.assetName, assetName);

  assertSmokeCommands((args) =>
    run("gig", args, {
      env: {
        PATH: `${globalPrefixBinDir(prefixDir)}${path.delimiter}${process.env.PATH || ""}`
      },
      shell: isWindows
    })
  );
});
