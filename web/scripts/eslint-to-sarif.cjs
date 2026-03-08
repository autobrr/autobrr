#!/usr/bin/env node

const path = require("node:path");
const { ESLint } = require("eslint");

const SARIF_SCHEMA = "https://json.schemastore.org/sarif-2.1.0.json";
const EMBED_SNIPPETS = process.env.SARIF_ESLINT_EMBED === "true";

function toPosix(filePath) {
  return filePath.split(path.sep).join("/");
}

function getRelativeUri(cwd, filePath) {
  const relativePath = path.relative(cwd, filePath);
  return toPosix(relativePath || path.basename(filePath));
}

function getSnippet(source, startLine, endLine) {
  if (!EMBED_SNIPPETS || !source) {
    return undefined;
  }

  const lines = source.split(/\r?\n/u);
  return {
    text: lines.slice(startLine - 1, endLine).join("\n"),
  };
}

function normalizeRuleId(message) {
  return message.ruleId || (message.fatal ? "eslint/fatal" : "eslint");
}

function severityToLevel(severity) {
  if (severity === 2) {
    return "error";
  }

  if (severity === 1) {
    return "warning";
  }

  return "note";
}

function createRuleDescriptor(ruleId, meta) {
  const docs = meta?.docs ?? {};

  return {
    id: ruleId,
    name: ruleId,
    shortDescription: {
      text: docs.description || ruleId,
    },
    helpUri: docs.url,
  };
}

function createResult(cwd, result, message, ruleIndexes) {
  const ruleId = normalizeRuleId(message);
  const startLine = message.line || 1;
  const startColumn = message.column || 1;
  const endLine = message.endLine || startLine;
  const endColumn = message.endColumn || startColumn;
  const uri = getRelativeUri(cwd, result.filePath);

  return {
    ruleId,
    ruleIndex: ruleIndexes.get(ruleId),
    level: severityToLevel(message.severity),
    message: {
      text: message.message,
    },
    locations: [
      {
        physicalLocation: {
          artifactLocation: {
            uri,
          },
          region: {
            startLine,
            startColumn,
            endLine,
            endColumn,
            snippet: getSnippet(result.source, startLine, endLine),
          },
        },
      },
    ],
  };
}

async function main() {
  const cwd = process.cwd();
  const targets = process.argv.slice(2);
  const files = targets.length > 0 ? targets : ["src/"];
  const eslint = new ESLint({ cwd });
  const results = await eslint.lintFiles(files);
  const rulesMeta = eslint.getRulesMetaForResults(results);
  const ruleIds = Array.from(
    new Set(results.flatMap((result) => result.messages.map(normalizeRuleId)))
  ).sort();
  const rules = ruleIds.map((ruleId) => createRuleDescriptor(ruleId, rulesMeta[ruleId]));
  const ruleIndexes = new Map(rules.map((rule, index) => [rule.id, index]));

  const sarif = {
    $schema: SARIF_SCHEMA,
    version: "2.1.0",
    runs: [
      {
        tool: {
          driver: {
            name: "ESLint",
            informationUri: "https://eslint.org",
            rules,
          },
        },
        results: results.flatMap((result) =>
          result.messages.map((message) => createResult(cwd, result, message, ruleIndexes))
        ),
      },
    ],
  };

  process.stdout.write(`${JSON.stringify(sarif, null, 2)}\n`);

  const hasErrors = results.some((result) => result.errorCount > 0 || result.fatalErrorCount > 0);
  process.exitCode = hasErrors ? 1 : 0;
}

main().catch((error) => {
  console.error(error);
  process.exit(2);
});
