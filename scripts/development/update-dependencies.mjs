#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { exit, stdin as input, stdout as output } from 'node:process';
import readline from 'node:readline/promises';

const ROOT_DIR = path.resolve(import.meta.dirname, '..', '..');
const BACKEND_DIR = path.join(ROOT_DIR, 'backend');
const TEMP_DIR = mkdtempSync(path.join(tmpdir(), 'pocket-id-deps-'));

const CONFIG = {
  minimumReleaseAgeDays: 7,
  snykArgs: [
    'test',
    '--all-projects',
    '--detection-depth=1',
    '--severity-threshold=medium',
  ],
  pnpmProjects: [
    { dir: '.', label: 'root workspace' },
    { dir: 'frontend', label: 'frontend' },
    { dir: 'tests', label: 'tests' },
    { dir: 'email-templates', label: 'email-templates' },
  ],
};

const ASSUME_YES = process.argv.includes('--yes') || process.argv.includes('-y');
const RELEASE_CUTOFF_MS = Date.now() - CONFIG.minimumReleaseAgeDays * 24 * 60 * 60 * 1000;
const COLOR = {
  red: '\x1b[31m',
  reset: '\x1b[0m',
};

const packagePublishTimelineCache = new Map();
const packagePublishTimeCache = new Map();
const goModuleVersionTimeCache = new Map();
const goModuleVersionsCache = new Map();

process.on('exit', () => {
  rmSync(TEMP_DIR, { recursive: true, force: true });
});

function printSection(title) {
  console.log(`\n== ${title} ==`);
}

function formatDate(value) {
  return new Date(value).toISOString();
}

function withColor(text, color) {
  return output.isTTY ? `${color}${text}${COLOR.reset}` : text;
}

function parseMajor(version) {
  const match = String(version).trim().replace(/^[^\d]*/, '').match(/^(\d+)/);
  return match ? Number(match[1]) : null;
}

function isMajorUpgrade(currentVersion, nextVersion) {
  const currentMajor = parseMajor(currentVersion);
  const nextMajor = parseMajor(nextVersion);
  return currentMajor !== null && nextMajor !== null && nextMajor > currentMajor;
}

function isSameMajorLine(currentVersion, candidateVersion) {
  const currentMajor = parseMajor(currentVersion);
  const candidateMajor = parseMajor(candidateVersion);
  return currentMajor !== null && currentMajor === candidateMajor;
}

function highlightUpgrade(text, currentVersion, nextVersion) {
  return isMajorUpgrade(currentVersion, nextVersion) ? withColor(text, COLOR.red) : text;
}

function isOlderThanMinimumAge(releaseTime) {
  return new Date(releaseTime).getTime() <= RELEASE_CUTOFF_MS;
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? ROOT_DIR,
    encoding: 'utf8',
    stdio: options.stdio ?? ['ignore', 'pipe', 'pipe'],
  });

  if (result.error) {
    throw new Error(`Failed to run '${command}': ${result.error.message}`);
  }

  return result;
}

function requireCommand(command) {
  const result = run('bash', ['-lc', `command -v ${command}`]);
  if (result.status !== 0) {
    throw new Error(`Required command '${command}' is not installed.`);
  }
}

function parseJson(stdout, errorContext) {
  try {
    return JSON.parse(stdout);
  } catch {
    throw new Error(`Failed to parse JSON for ${errorContext}.`);
  }
}

function partitionRows(rows, predicate) {
  const eligibleRows = [];
  const heldBackRows = [];

  for (const row of rows) {
    if (predicate(row)) {
      eligibleRows.push(row);
    } else {
      heldBackRows.push(row);
    }
  }

  return { eligibleRows, heldBackRows };
}

function parsePnpmOutdated(rawText) {
  if (!rawText.trim()) {
    return [];
  }

  const raw = parseJson(rawText, 'pnpm outdated output');

  const collectRows = (value) => {
    if (!value || typeof value !== 'object') return [];
    if (Array.isArray(value)) return value.flatMap(collectRows);
    if (Array.isArray(value.results)) return collectRows(value.results);
    if ('current' in value || 'latest' in value || 'wanted' in value) return [value];

    return Object.entries(value).flatMap(([name, info]) => {
      if (Array.isArray(info)) {
        return info.flatMap((entry) => collectRows({ name, ...entry }));
      }
      if (info && typeof info === 'object') {
        return collectRows({ name, ...info });
      }
      return [];
    });
  };

  return collectRows(raw)
    .map((row) => ({
      name: row.name || row.package || row.packageName || 'unknown',
      current: row.current || 'unknown',
      latest: row.latest || row.wanted || 'unknown',
      type: row.dependencyType || row.type || '',
    }))
    .filter((row) => row.current !== row.latest)
    .sort((left, right) => left.name.localeCompare(right.name));
}

function getPnpmPackagePublishTime(name, version) {
  const cacheKey = `${name}@${version}`;
  const cachedPublishTime = packagePublishTimeCache.get(cacheKey);
  if (cachedPublishTime) {
    return cachedPublishTime;
  }

  let timeline = packagePublishTimelineCache.get(name);
  if (!timeline) {
    const result = run('pnpm', ['view', name, 'time', '--json']);
    if (result.status !== 0 || !result.stdout.trim()) {
      throw new Error(`Failed to get publish timeline for ${name}\n${result.stderr}`.trim());
    }

    timeline = parseJson(result.stdout, `pnpm publish timeline for ${name}`);
    if (!timeline || typeof timeline !== 'object') {
      throw new Error(`Publish timeline for ${name} is unavailable.`);
    }

    packagePublishTimelineCache.set(name, timeline);
  }

  const publishTime = timeline[version];
  if (!publishTime) {
    throw new Error(`Publish time for ${cacheKey} is unavailable.`);
  }

  packagePublishTimeCache.set(cacheKey, publishTime);
  return publishTime;
}

function getPnpmUpgradeSummary(project) {
  const result = run('pnpm', ['outdated', '--format', 'json'], {
    cwd: path.join(ROOT_DIR, project.dir),
  });

  if (result.status !== 0 && !result.stdout.trim()) {
    throw new Error(`${project.label}: failed to collect pnpm updates\n${result.stderr}`.trim());
  }

  const rows = parsePnpmOutdated(result.stdout).map((row) => ({
    ...row,
    publishTime: getPnpmPackagePublishTime(row.name, row.latest),
  }));

  return {
    project,
    ...partitionRows(rows, (row) => isOlderThanMinimumAge(row.publishTime)),
  };
}

function parseGoListJsonStream(rawText) {
  const objects = [];
  let depth = 0;
  let start = -1;
  let inString = false;
  let escaped = false;

  for (let index = 0; index < rawText.length; index += 1) {
    const char = rawText[index];

    if (escaped) {
      escaped = false;
      continue;
    }

    if (char === '\\') {
      escaped = true;
      continue;
    }

    if (char === '"') {
      inString = !inString;
      continue;
    }

    if (inString) {
      continue;
    }

    if (char === '{') {
      if (depth === 0) {
        start = index;
      }
      depth += 1;
      continue;
    }

    if (char === '}') {
      depth -= 1;
      if (depth === 0 && start !== -1) {
        objects.push(parseJson(rawText.slice(start, index + 1), 'go module JSON stream'));
        start = -1;
      }
    }
  }

  return objects;
}

function getGoModuleVersionTime(name, version) {
  const cacheKey = `${name}@${version}`;
  const cachedTime = goModuleVersionTimeCache.get(cacheKey);
  if (cachedTime) {
    return cachedTime;
  }

  const result = run('go', ['list', '-m', '-json', `${name}@${version}`], {
    cwd: BACKEND_DIR,
  });

  if (result.status !== 0 || !result.stdout.trim()) {
    throw new Error(`Failed to get release time for Go module ${cacheKey}\n${result.stderr}`.trim());
  }

  const moduleInfo = parseJson(result.stdout, `Go module ${cacheKey}`);
  if (!moduleInfo.Time) {
    throw new Error(`Release time for Go module ${cacheKey} is unavailable.`);
  }

  goModuleVersionTimeCache.set(cacheKey, moduleInfo.Time);
  return moduleInfo.Time;
}

function getGoModuleVersions(name) {
  const cachedVersions = goModuleVersionsCache.get(name);
  if (cachedVersions) {
    return cachedVersions;
  }

  const result = run('go', ['list', '-m', '-versions', '-json', name], {
    cwd: BACKEND_DIR,
  });

  if (result.status !== 0 || !result.stdout.trim()) {
    throw new Error(`Failed to get versions for Go module ${name}\n${result.stderr}`.trim());
  }

  const moduleInfo = parseJson(result.stdout, `Go module versions for ${name}`);
  const versions = Array.isArray(moduleInfo.Versions) ? moduleInfo.Versions : [];
  goModuleVersionsCache.set(name, versions);
  return versions;
}

function resolveGoUpgradeTarget(name, currentVersion) {
  const versions = getGoModuleVersions(name);
  const currentVersionIndex = versions.indexOf(currentVersion);

  const candidateVersions = (currentVersionIndex === -1 ? versions : versions.slice(currentVersionIndex + 1))
    .filter((version) => isSameMajorLine(currentVersion, version));

  if (candidateVersions.length === 0) {
    return null;
  }

  const latest = candidateVersions.at(-1);
  const latestPublishTime = getGoModuleVersionTime(name, latest);

  for (let index = candidateVersions.length - 1; index >= 0; index -= 1) {
    const target = candidateVersions[index];
    const targetPublishTime = getGoModuleVersionTime(name, target);

    if (isOlderThanMinimumAge(targetPublishTime)) {
      return {
        latest,
        latestPublishTime,
        target,
        targetPublishTime,
      };
    }
  }

  return {
    latest,
    latestPublishTime,
    target: null,
    targetPublishTime: null,
  };
}

function getGoUpgradeSummary() {
  const result = run('go', ['list', '-m', '-u', '-json', 'all'], {
    cwd: BACKEND_DIR,
  });

  if (result.status !== 0) {
    throw new Error(`backend/go.mod: failed to collect Go updates\n${result.stderr}`.trim());
  }

  const rows = parseGoListJsonStream(result.stdout)
    .filter((moduleInfo) => !moduleInfo.Main && moduleInfo.Update?.Version && moduleInfo.Version && !moduleInfo.Indirect)
    .map((moduleInfo) => {
      const resolution = resolveGoUpgradeTarget(moduleInfo.Path, moduleInfo.Version);
      if (!resolution) {
        return null;
      }

      return {
        name: moduleInfo.Path,
        current: moduleInfo.Version,
        ...resolution,
      };
    })
    .filter(Boolean)
    .sort((left, right) => left.name.localeCompare(right.name));

  return partitionRows(rows, (row) => Boolean(row.target));
}

function formatPnpmRow(row, statusLabel) {
  const suffix = row.type ? ` (${row.type})` : '';
  return highlightUpgrade(
    `  - ${row.name}: ${row.current} -> ${row.latest}${suffix} [${statusLabel}, published ${formatDate(row.publishTime)}]`,
    row.current,
    row.latest
  );
}

function printPnpmSummaries(summaries) {
  printSection(`Planned pnpm Upgrades (minimum age: ${CONFIG.minimumReleaseAgeDays} days)`);

  for (const { project, eligibleRows, heldBackRows } of summaries) {
    if (eligibleRows.length === 0 && heldBackRows.length === 0) {
      console.log(`${project.label}: no pnpm upgrades available`);
      continue;
    }

    console.log(
      `${project.label}: ${eligibleRows.length} eligible pnpm upgrade(s), ${heldBackRows.length} held back`
    );

    for (const row of eligibleRows) {
      console.log(formatPnpmRow(row, 'eligible'));
    }

    for (const row of heldBackRows) {
      console.log(formatPnpmRow(row, 'held back'));
    }
  }
}

function formatGoRow(row) {
  const details = row.target === row.latest
    ? `[eligible, published ${formatDate(row.targetPublishTime)}]`
    : `[eligible fallback, latest ${row.latest} published ${formatDate(row.latestPublishTime)}, selected ${row.target} published ${formatDate(row.targetPublishTime)}]`;

  return highlightUpgrade(
    `  - ${row.name}: ${row.current} -> ${row.target} ${details}`,
    row.current,
    row.target
  );
}

function formatHeldBackGoRow(row) {
  return highlightUpgrade(
    `  - ${row.name}: ${row.current} -> ${row.latest} [held back, latest published ${formatDate(row.latestPublishTime)}]`,
    row.current,
    row.latest
  );
}

function printGoSummary(summary) {
  printSection(`Planned Go Upgrades (minimum age: ${CONFIG.minimumReleaseAgeDays} days)`);

  if (summary.eligibleRows.length === 0 && summary.heldBackRows.length === 0) {
    console.log('backend/go.mod: no Go upgrades available');
    return;
  }

  console.log(
    `backend/go.mod: ${summary.eligibleRows.length} eligible Go upgrade(s), ${summary.heldBackRows.length} held back`
  );

  for (const row of summary.eligibleRows) {
    console.log(formatGoRow(row));
  }

  for (const row of summary.heldBackRows) {
    console.log(formatHeldBackGoRow(row));
  }
}

function parseSnykResults(rawText) {
  const raw = parseJson(rawText, 'Snyk results');
  const results = Array.isArray(raw) ? raw : Array.isArray(raw.results) ? raw.results : [raw];

  const totals = { critical: 0, high: 0, medium: 0 };
  let projects = 0;
  let affectedProjects = 0;

  for (const result of results) {
    if (!result || typeof result !== 'object') {
      continue;
    }

    projects += 1;

    const vulnerabilities = Array.isArray(result.vulnerabilities)
      ? result.vulnerabilities
      : Array.isArray(result.issues?.vulnerabilities)
        ? result.issues.vulnerabilities
        : [];

    if (vulnerabilities.length > 0) {
      affectedProjects += 1;
    }

    for (const vulnerability of vulnerabilities) {
      const severity = String(vulnerability.severity || '').toLowerCase();
      if (severity in totals) {
        totals[severity] += 1;
      }
    }
  }

  return { totals, projects, affectedProjects };
}

function printSnykStatus(label) {
  console.log(`\nCollecting Snyk vulnerability status for ${label.toLowerCase()}...`);
  
  const outputFile = path.join(TEMP_DIR, `${label.toLowerCase().replaceAll(' ', '-')}.json`);
  const result = run('snyk', [...CONFIG.snykArgs, `--json-file-output=${outputFile}`]);

  if (![0, 1].includes(result.status)) {
    throw new Error(`${label}: failed to collect Snyk status\n${result.stderr}`.trim());
  }

  const summary = parseSnykResults(readFileSync(outputFile, 'utf8'));

  printSection(`${label} Vulnerability Status`);
  console.log(`${label}: ${summary.projects} project scan(s), ${summary.affectedProjects} with vulnerabilities`);
  console.log(`  - critical: ${summary.totals.critical}`);
  console.log(`  - high: ${summary.totals.high}`);
  console.log(`  - medium: ${summary.totals.medium}`);
  console.log(`  - total: ${Object.values(summary.totals).reduce((sum, count) => sum + count, 0)}`);

  return result.status;
}

async function confirmUpgrade() {
  if (ASSUME_YES) {
    return;
  }

  const rl = readline.createInterface({ input, output });
  const answer = await rl.question('\nProceed with dependency upgrades? (y/n) ');
  rl.close();

  if (answer !== 'y') {
    throw new Error('Dependency upgrade canceled.');
  }
}

function applyPnpmUpgrades() {
  printSection('Applying pnpm Upgrades');

  const result = run('pnpm', ['update', '-r', '--latest'], { stdio: 'inherit' });
  if (result.status !== 0) {
    throw new Error('pnpm workspace upgrade failed.');
  }
}

function applyGoUpgrades(goSummary) {
  printSection('Applying Go Upgrades');

  if (goSummary.eligibleRows.length === 0) {
    console.log(`No Go upgrades met the ${CONFIG.minimumReleaseAgeDays}-day minimum age.`);
  } else {
    const moduleSpecs = goSummary.eligibleRows.map((row) => `${row.name}@${row.target}`);
    const result = run('go', ['get', '-t', ...moduleSpecs], {
      cwd: BACKEND_DIR,
      stdio: 'inherit',
    });

    if (result.status !== 0) {
      throw new Error('Go dependency upgrade failed.');
    }
  }

  const tidyResult = run('go', ['mod', 'tidy'], {
    cwd: BACKEND_DIR,
    stdio: 'inherit',
  });

  if (tidyResult.status !== 0) {
    throw new Error('go mod tidy failed.');
  }
}

async function main() {
  requireCommand('pnpm');
  requireCommand('go');
  requireCommand('snyk');

  const pnpmSummaries = CONFIG.pnpmProjects.map(getPnpmUpgradeSummary);
  const goSummary = getGoUpgradeSummary();

  printPnpmSummaries(pnpmSummaries);
  printGoSummary(goSummary);
  printSnykStatus('Before Upgrade');

  const hasEligibleUpgrades =
    pnpmSummaries.some((summary) => summary.eligibleRows.length > 0) ||
    goSummary.eligibleRows.length > 0;

  if (!hasEligibleUpgrades) {
    console.log(`\nNo dependency upgrades met the ${CONFIG.minimumReleaseAgeDays}-day minimum age.`);
    exit(printSnykStatus('After Upgrade'));
  }

  await confirmUpgrade();
  applyPnpmUpgrades();
  applyGoUpgrades(goSummary);

  exit(printSnykStatus('After Upgrade'));
}

main().catch((error) => {
  console.error(error.message);
  exit(1);
});
