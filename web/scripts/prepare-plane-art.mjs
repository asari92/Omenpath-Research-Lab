import { createHash } from "node:crypto";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

import sharp from "sharp";
import { auditPlaneArt } from "./audit-plane-art.mjs";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(scriptDir, "..");
const repositoryRoot = path.resolve(webRoot, "..");
const seedPath = path.join(
  repositoryRoot,
  "data",
  "mtg_planes_expanded_seed.json",
);
const sourcesPath = path.join(
  repositoryRoot,
  "data",
  "plane_image_sources.json",
);
const manifestPath = path.join(
  repositoryRoot,
  "data",
  "plane_images_manifest.json",
);
const outputDir = path.join(webRoot, "public", "planes");
const policyURL = "https://company.wizards.com/en/legal/fancontentpolicy";
const scryfallFAQ =
  "https://scryfall.com/docs/faqs/i-m-having-trouble-accessing-the-scryfall-api-or-i-m-blocked-17";
const userAgent =
  "OmenpathResearchLab/1.0 (local educational portfolio project)";

const mode = process.argv.includes("--discover")
  ? "discover"
  : process.argv.includes("--offline")
    ? "offline"
    : null;

if (!mode) throw new Error("Use --discover or --offline");

const seed = JSON.parse(await readFile(seedPath, "utf8"));
await mkdir(outputDir, { recursive: true });

if (mode === "offline") {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
  if (manifest.entries.length !== seed.planes.length) {
    throw new Error("Offline manifest does not cover every seeded Plane");
  }
  for (const entry of manifest.entries) {
    const file = await readFile(
      path.join(webRoot, "public", entry.local_path.replace(/^\//, "")),
    );
    if (sha256(file) !== entry.sha256)
      throw new Error(`Hash mismatch for Plane ${entry.plane_id}`);
  }
  const audit = await auditPlaneArt();
  if (audit.errors.length) throw new Error(audit.errors.join("\n"));
  process.stdout.write(
    `Verified ${manifest.entries.length} local Plane assets offline.\n`,
  );
  process.exit(0);
}

const cards = await discoverPlaneCards();
const bySubtype = new Map();
for (const card of cards) {
  const subtype = planeSubtype(card.type_line);
  if (!subtype) continue;
  const key = normalize(subtype);
  const prior = bySubtype.get(key);
  if (!prior || card.id.localeCompare(prior.id) < 0) bySubtype.set(key, card);
}

const sourceEntries = [];
const manifestEntries = [];
for (const plane of seed.planes) {
  const names = [plane.name, ...plane.aliases].map(normalize);
  const card = names.map((name) => bySubtype.get(name)).find(Boolean) ?? null;
  const slug = slugify(plane.name);
  const filename = `${String(plane.id).padStart(2, "0")}-${slug}.webp`;
  const localPath = `/planes/${filename}`;
  let input;
  let source;

  if (card && cardImage(card)) {
    const imageURL = cardImage(card);
    input = await downloadImage(imageURL);
    source = {
      plane_id: plane.id,
      plane_name: plane.name,
      source_url: card.scryfall_uri,
      source_object_id: card.id,
      image_url: imageURL,
      credit: `${card.artist ?? "Unknown artist"} — ${card.name}`,
      policy_url: policyURL,
      fallback: false,
    };
  } else {
    input = Buffer.from(fallbackSVG(plane.name, plane.id));
    source = {
      plane_id: plane.id,
      plane_name: plane.name,
      source_url: "generated:omenpath-plane-fallback",
      source_object_id: null,
      image_url: null,
      credit: "Omenpath Research Lab generated fallback",
      policy_url: policyURL,
      fallback: true,
    };
  }

  const webp = await boundedWebP(input);
  await writeFile(path.join(outputDir, filename), webp);
  sourceEntries.push(source);
  manifestEntries.push({
    plane_id: plane.id,
    plane_name: plane.name,
    local_path: localPath,
    source_url: source.source_url,
    source_object_id: source.source_object_id,
    credit: source.credit,
    policy_url: source.policy_url,
    sha256: sha256(webp),
    fallback: source.fallback,
  });
  await delay(105);
}

const generic = await boundedWebP(Buffer.from(fallbackSVG("Unknown Plane", 0)));
await writeFile(path.join(outputDir, "fallback.webp"), generic);

await writeJSON(sourcesPath, {
  source: "Scryfall Plane cards",
  api_guidance: scryfallFAQ,
  entries: sourceEntries,
});
await writeJSON(manifestPath, {
  fan_content_notice:
    "Omenpath Research Lab is unofficial fan content and is not approved or endorsed by Wizards.",
  policy_url: policyURL,
  generic_fallback_path: "/planes/fallback.webp",
  entries: manifestEntries,
});

process.stdout.write(
  `Prepared ${manifestEntries.length} Plane assets (${manifestEntries.filter((entry) => entry.fallback).length} fallbacks).\n`,
);

async function discoverPlaneCards() {
  const result = [];
  let next =
    "https://api.scryfall.com/cards/search?q=t%3Aplane&unique=cards&order=name";
  while (next) {
    const response = await fetch(next, {
      headers: { Accept: "application/json", "User-Agent": userAgent },
      redirect: "error",
    });
    if (!response.ok)
      throw new Error(`Scryfall request failed: ${response.status}`);
    const page = await response.json();
    result.push(...page.data);
    next = page.has_more ? page.next_page : null;
    if (next) await delay(105);
  }
  return result;
}

function cardImage(card) {
  return (
    card.image_uris?.art_crop ??
    card.card_faces?.find((face) => face.image_uris)?.image_uris.art_crop
  );
}

async function downloadImage(url) {
  const response = await fetch(url, {
    headers: { Accept: "image/*", "User-Agent": userAgent },
    redirect: "error",
  });
  if (!response.ok) throw new Error(`Image request failed: ${response.status}`);
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.toLowerCase().startsWith("image/")) {
    throw new Error(`Rejected non-image response: ${contentType}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

async function boundedWebP(input) {
  for (const quality of [78, 70, 62, 54, 46, 38]) {
    const output = await sharp(input)
      .resize(512, 512, { fit: "cover", position: "attention" })
      .webp({ quality, effort: 4 })
      .toBuffer();
    if (output.byteLength <= 180 * 1024) return output;
  }
  throw new Error("Unable to fit artwork below 180 KiB");
}

function planeSubtype(typeLine = "") {
  const parts = typeLine.split("—");
  return parts.length === 2 && normalize(parts[0]) === "plane"
    ? parts[1].trim()
    : null;
}

function normalize(value) {
  return value
    .normalize("NFKD")
    .replace(/[’']/g, "")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim()
    .toLowerCase();
}

function slugify(value) {
  return normalize(value).replaceAll(" ", "-");
}

function fallbackSVG(name, id) {
  const hue = (id * 47 + 168) % 360;
  const safe = name.replace(
    /[&<>"']/g,
    (character) =>
      ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': "&quot;",
        "'": "&apos;",
      })[character],
  );
  return `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512"><defs><radialGradient id="g"><stop stop-color="hsl(${hue} 75% 43%)"/><stop offset="1" stop-color="#050912"/></radialGradient></defs><rect width="512" height="512" fill="url(#g)"/><circle cx="256" cy="256" r="188" fill="none" stroke="hsla(${hue} 95% 72% / .35)" stroke-width="3"/><path d="M40 330 Q155 180 270 315 T500 260" fill="none" stroke="hsla(${hue} 90% 80% / .24)" stroke-width="24"/><text x="256" y="455" text-anchor="middle" fill="#eefcff" font-family="sans-serif" font-size="28">${safe}</text></svg>`;
}

function sha256(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

async function writeJSON(target, value) {
  await writeFile(target, `${JSON.stringify(value, null, 2)}\n`);
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
