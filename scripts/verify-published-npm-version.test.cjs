#!/usr/bin/env node
"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawnSync } = require("node:child_process");

const repoRoot = path.resolve(__dirname, "..");
const verifyScript = path.join(repoRoot, "scripts", "verify-published-npm-version.sh");

async function withFakeNpm(scenario, fn) {
  const tempRoot = await fs.promises.mkdtemp(path.join(os.tmpdir(), "gig-verify-npm-"));
  const logPath = path.join(tempRoot, "npm-calls.log");
  const statePath = path.join(tempRoot, "npm-state.txt");
  const fakeNpmPath = path.join(tempRoot, "npm");

  const fakeNpmSource = `#!/usr/bin/env node
"use strict";
const fs = require("node:fs");

const scenario = process.env.FAKE_NPM_SCENARIO;
const logPath = process.env.FAKE_NPM_LOG;
const statePath = process.env.FAKE_NPM_STATE;
const args = process.argv.slice(2);

if (logPath) {
  fs.appendFileSync(logPath, JSON.stringify(args) + "\\n");
}

if (args[0] !== "view") {
  console.error("unexpected npm command: " + args.join(" "));
  process.exit(1);
}

const packageArg = args[1];
const expectedVersion = "2026.4.20";

if (scenario === "exact-version-visible") {
  if (packageArg === "@hunpeolabs/gig@" + expectedVersion) {
    process.stdout.write(expectedVersion + "\\n");
    process.exit(0);
  }

  if (packageArg === "@hunpeolabs/gig") {
    process.stdout.write("2026.4.14\\n");
    process.exit(0);
  }

  console.error("unexpected package lookup: " + packageArg);
  process.exit(1);
}

if (scenario === "version-404-then-visible") {
  if (packageArg !== "@hunpeolabs/gig@" + expectedVersion) {
    console.error("unexpected package lookup: " + packageArg);
    process.exit(1);
  }

  let attempt = 0;
  if (fs.existsSync(statePath)) {
    attempt = Number(fs.readFileSync(statePath, "utf8")) || 0;
  }
  attempt += 1;
  fs.writeFileSync(statePath, String(attempt));

  if (attempt < 3) {
    process.stderr.write("npm ERR! code E404\\nnpm ERR! 404 Not Found\\n");
    process.exit(1);
  }

  process.stdout.write(expectedVersion + "\\n");
  process.exit(0);
}

console.error("unexpected scenario: " + scenario);
process.exit(1);
`;

  await fs.promises.writeFile(fakeNpmPath, fakeNpmSource, { mode: 0o755 });

  try {
    return await fn({
      logPath,
      statePath,
      pathEnv: `${tempRoot}${path.delimiter}${process.env.PATH || ""}`
    });
  } finally {
    await fs.promises.rm(tempRoot, { recursive: true, force: true });
  }
}

function readCalls(logPath) {
  if (!fs.existsSync(logPath)) {
    return [];
  }

  return fs
    .readFileSync(logPath, "utf8")
    .trim()
    .split(/\n+/)
    .filter(Boolean)
    .map((line) => JSON.parse(line));
}

function runVerifier(pathEnv, extraEnv = {}) {
  return spawnSync("bash", [verifyScript, "@hunpeolabs/gig", "v2026.04.20"], {
    cwd: repoRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: pathEnv,
      NPM_VERIFY_MAX_ATTEMPTS: "5",
      NPM_VERIFY_SLEEP_SECONDS: "0",
      ...extraEnv
    }
  });
}

test("verify script checks the exact published npm version", async () => {
  await withFakeNpm("exact-version-visible", async ({ logPath, pathEnv, statePath }) => {
    const result = runVerifier(pathEnv, {
      FAKE_NPM_SCENARIO: "exact-version-visible",
      FAKE_NPM_LOG: logPath,
      FAKE_NPM_STATE: statePath
    });

    assert.equal(result.status, 0, result.stderr || result.stdout);
    assert.match(result.stdout, /Published @hunpeolabs\/gig@2026\.4\.20/);

    const calls = readCalls(logPath);
    assert.equal(calls.length, 1);
    assert.deepEqual(calls[0], [
      "view",
      "@hunpeolabs/gig@2026.4.20",
      "version",
      "--registry=https://registry.npmjs.org/"
    ]);
  });
});

test("verify script retries until the exact npm version becomes visible", async () => {
  await withFakeNpm("version-404-then-visible", async ({ logPath, pathEnv, statePath }) => {
    const result = runVerifier(pathEnv, {
      FAKE_NPM_SCENARIO: "version-404-then-visible",
      FAKE_NPM_LOG: logPath,
      FAKE_NPM_STATE: statePath
    });

    assert.equal(result.status, 0, result.stderr || result.stdout);
    assert.match(result.stderr, /not visible on npm yet/i);

    const calls = readCalls(logPath);
    assert.equal(calls.length, 3);
    for (const call of calls) {
      assert.deepEqual(call, [
        "view",
        "@hunpeolabs/gig@2026.4.20",
        "version",
        "--registry=https://registry.npmjs.org/"
      ]);
    }
  });
});
