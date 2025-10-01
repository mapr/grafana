#!/usr/bin/env node

const fs = require('node:fs');
const { stat } = require('node:fs/promises');
const readline = require('node:readline');
const Stream = require('node:stream');

const {
  RAW_AUDIT_JSONL_PATH,
  CODEOWNERS_BY_FILENAME_JSON_PATH,
  FILENAMES_BY_CODEOWNER_JSON_PATH,
  CODEOWNERS_JSON_PATH,
} = require('./constants.js');

/**
 * @param {Stream} stream
 * @param {string} name
 */
const logStreamError = (stream, name) => {
  stream.on('error', (err) => {
    console.error(`Stream error | ${name}`, err);
  });
};

/**
 * Generate codeowners manifest files from raw audit data
 * @param {string} rawAuditPath - Path to the raw audit JSONL file
 * @param {string} codeownersJsonPath - Path to write teams.json
 * @param {string} codeownersByFilenamePath - Path to write teams-by-filename.json
 * @param {string} filenamesByCodeownerPath - Path to write filenames-by-team.json
 */
async function generateCodeownersManifest(
  rawAuditPath,
  codeownersJsonPath,
  codeownersByFilenamePath,
  filenamesByCodeownerPath
) {
  const hasRawAuditJsonl = await stat(rawAuditPath);
  if (!hasRawAuditJsonl) {
    throw new Error(
      `No raw CODEOWNERS audit JSONL file found at: ${rawAuditPath} ... run "yarn codeowners-manifest:raw"`
    );
  }

  const auditFileInput = fs.createReadStream(rawAuditPath);

  const codeownersOutput = fs.createWriteStream(codeownersJsonPath);
  const codeownersByFilenameOutput = fs.createWriteStream(codeownersByFilenamePath);
  const filenamesByCodeownerOutput = fs.createWriteStream(filenamesByCodeownerPath);

  logStreamError(auditFileInput, 'AuditFileRead');
  logStreamError(codeownersOutput, 'CodeownersWrite');
  logStreamError(codeownersByFilenameOutput, 'CodeownersByFilenameWrite');
  logStreamError(filenamesByCodeownerOutput, 'FilenamesByCodeownerWrie');

  const lineReader = readline.createInterface({
    input: auditFileInput,
    crlfDelay: Infinity,
  });

  let codeowners = new Set();
  let codeownersByFilename = {};
  let filenamesByCodeowner = {};

  lineReader.on('line', (line) => {
    const { path, owners: fileOwners } = JSON.parse(line.toString().trim());

    for (let owner of fileOwners) {
      codeowners.add(owner);
    }

    codeownersByFilename[path] = fileOwners;

    for (let owner of fileOwners) {
      const filenames = filenamesByCodeowner[owner] || [];
      filenamesByCodeowner[owner] = filenames.concat(path);
    }
  });

  await new Promise((resolve) => lineReader.once('close', resolve));

  codeownersOutput.write(JSON.stringify(Array.from(codeowners).sort(), null, 2));
  codeownersByFilenameOutput.write(JSON.stringify(codeownersByFilename, null, 2));
  filenamesByCodeownerOutput.write(JSON.stringify(filenamesByCodeowner, null, 2));
}

if (require.main === module) {
  (async () => {
    try {
      console.log(`📋 Generating files ↔ teams manifests from ${RAW_AUDIT_JSONL_PATH} ...`);
      await generateCodeownersManifest(
        RAW_AUDIT_JSONL_PATH,
        CODEOWNERS_JSON_PATH,
        CODEOWNERS_BY_FILENAME_JSON_PATH,
        FILENAMES_BY_CODEOWNER_JSON_PATH
      );
      console.log('✅ Manifest files generated:');
      console.log(`   • ${CODEOWNERS_JSON_PATH}`);
      console.log(`   • ${CODEOWNERS_BY_FILENAME_JSON_PATH}`);
      console.log(`   • ${FILENAMES_BY_CODEOWNER_JSON_PATH}`);
    } catch (e) {
      console.error(e);
      process.exit(1);
    }
  })();
}

module.exports = { generateCodeownersManifest };
