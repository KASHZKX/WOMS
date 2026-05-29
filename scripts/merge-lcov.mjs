#!/usr/bin/env node
import { readFileSync, writeFileSync } from "node:fs";

const file = process.argv[2];
if (!file) {
  console.error("Usage: merge-lcov.mjs <lcov.info>");
  process.exit(1);
}

const records = readFileSync(file, "utf8")
  .split("end_of_record")
  .map((record) => record.trim())
  .filter(Boolean);

const bySource = new Map();

function sourceRecord(source) {
  if (!bySource.has(source)) {
    bySource.set(source, {
      source,
      functions: new Map(),
      functionHits: new Map(),
      branches: new Map(),
      lines: new Map(),
    });
  }
  return bySource.get(source);
}

for (const recordText of records) {
  const lines = recordText.split(/\r?\n/);
  const sourceLine = lines.find((line) => line.startsWith("SF:"));
  if (!sourceLine) {
    continue;
  }
  const record = sourceRecord(sourceLine.slice(3));
  for (const line of lines) {
    if (line.startsWith("FN:")) {
      const [lineNo, name] = line.slice(3).split(",", 2);
      record.functions.set(name, lineNo);
      continue;
    }
    if (line.startsWith("FNDA:")) {
      const [hits, name] = line.slice(5).split(",", 2);
      record.functionHits.set(name, (record.functionHits.get(name) ?? 0) + Number(hits));
      continue;
    }
    if (line.startsWith("BRDA:")) {
      const parts = line.slice(5).split(",");
      const key = parts.slice(0, 3).join(",");
      const hits = parts[3] === "-" ? 0 : Number(parts[3]);
      record.branches.set(key, (record.branches.get(key) ?? 0) + hits);
      continue;
    }
    if (line.startsWith("DA:")) {
      const [lineNo, hits] = line.slice(3).split(",", 2);
      record.lines.set(lineNo, (record.lines.get(lineNo) ?? 0) + Number(hits));
    }
  }
}

const output = [];
for (const record of bySource.values()) {
  output.push(`SF:${record.source}`);
  for (const [name, lineNo] of [...record.functions.entries()].sort((a, b) => Number(a[1]) - Number(b[1]) || a[0].localeCompare(b[0]))) {
    output.push(`FN:${lineNo},${name}`);
  }
  for (const [name, hits] of [...record.functionHits.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
    output.push(`FNDA:${hits},${name}`);
  }
  output.push(`FNF:${record.functions.size}`);
  output.push(`FNH:${[...record.functionHits.values()].filter((hits) => hits > 0).length}`);
  for (const [key, hits] of [...record.branches.entries()].sort((a, b) => {
    const left = a[0].split(",").map(Number);
    const right = b[0].split(",").map(Number);
    return left[0] - right[0] || left[1] - right[1] || left[2] - right[2];
  })) {
    output.push(`BRDA:${key},${hits}`);
  }
  output.push(`BRF:${record.branches.size}`);
  output.push(`BRH:${[...record.branches.values()].filter((hits) => hits > 0).length}`);
  for (const [lineNo, hits] of [...record.lines.entries()].sort((a, b) => Number(a[0]) - Number(b[0]))) {
    output.push(`DA:${lineNo},${hits}`);
  }
  output.push(`LF:${record.lines.size}`);
  output.push(`LH:${[...record.lines.values()].filter((hits) => hits > 0).length}`);
  output.push("end_of_record");
}

writeFileSync(file, `${output.join("\n")}\n`);
