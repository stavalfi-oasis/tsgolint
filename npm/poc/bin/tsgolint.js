#!/usr/bin/env node

// Oasis fork wrapper. Unlike upstream's npm/core wrapper, the platform binaries
// are bundled in this same package rather than resolved from optionalDependencies
// — rustfs serves static tarballs and cannot resolve a dependency tree.
// Node resolves the node_modules/.bin symlink before setting __dirname, so this
// points at the package's own bin/ directory. stdio is inherited, which keeps
// oxlint's stdin/stdout JSON protocol intact.

const process = require('node:process');
const child_process = require('node:child_process');
const path = require('node:path');

const exePath = path.join(__dirname, `tsgolint-${process.platform}-${process.arch}`);

try {
  child_process.execFileSync(exePath, process.argv.slice(2), { stdio: 'inherit' });
} catch (e) {
  if (e.status) {
    process.exitCode = e.status;
  } else {
    throw e;
  }
}
