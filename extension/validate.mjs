import assert from "node:assert/strict";
import {readFile} from "node:fs/promises";

const manifest = JSON.parse(await readFile(new URL("manifest.json", import.meta.url)));
const rules = JSON.parse(await readFile(new URL("rules/privacy-signals.json", import.meta.url)));

assert.equal(manifest.manifest_version, 3);
assert.ok(manifest.permissions.includes("declarativeNetRequestWithHostAccess"));
assert.deepEqual(manifest.host_permissions, ["<all_urls>"]);

const mainWorldScript = manifest.content_scripts.find(script => script.js?.includes("gpc.js"));
assert.equal(mainWorldScript?.world, "MAIN");
assert.equal(mainWorldScript?.run_at, "document_start");

const headers = rules.flatMap(rule => rule.action?.requestHeaders || []);
assert.ok(headers.some(header => header.header.toLowerCase() === "sec-gpc" && header.value === "1"));
assert.ok(headers.some(header => header.header.toLowerCase() === "dnt" && header.value === "1"));

console.log("Extension manifest and privacy signals are valid.");
