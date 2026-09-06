import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const json = async (name) => JSON.parse(await readFile(path.join(root, "data", name), "utf8"));
const rejected = new Map([
  [14, "634109c01fa423126e576f7eb26413c7a7ccd17e2605e56e5a24a100ccf7a303"],
  [21, "8099f4d45f160c205950506c0eacd82a59fb3757ee522e5ea645cfca9a62a52d"],
  [32, "6de433ae14a27bfbfa7a862742e898a7b66718b3ba4b1adcf90c286a80df0326"],
  [75, "878887e8bd371d73d174f691dc287ac5f18726e17f398ad33eba77884846a460"],
]);

export async function auditPlaneArt() {
  const [seed, manifest, sources, audit] = await Promise.all([
    json("mtg_planes_expanded_seed.json"), json("plane_images_manifest.json"),
    json("plane_image_sources.json"), json("plane_art_audit.json"),
  ]);
  const errors = [];
  const ids = seed.planes.map((p) => p.id).sort((a,b) => a-b);
  for (const [name, rows] of [["manifest",manifest.entries],["sources",sources.entries],["audit",audit]]) {
    if (rows.length !== 85 || JSON.stringify(rows.map((e)=>e.plane_id).sort((a,b)=>a-b)) !== JSON.stringify(ids)) errors.push(`${name}: expected exactly 85 unique seed IDs`);
  }
  const paths = new Set();
  const hashes = new Set();
  for (const entry of manifest.entries) {
    const fail = (message) => errors.push(`Plane ${entry.plane_id}: ${message}`);
    const review = audit.find((e) => e.plane_id === entry.plane_id);
    const source = sources.entries.find((e) => e.plane_id === entry.plane_id);
    if (!/^\/planes\/[a-z0-9-]+\.webp$/.test(entry.local_path) || paths.has(entry.local_path)) { fail("invalid or duplicate local runtime path"); continue; }
    paths.add(entry.local_path);
    if (!review || review.local_path !== entry.local_path || review.reviewed !== true || review.text_free !== true || review.frame_free !== true) fail("not reviewed text/frame-free");
    if (!review || !["fan_content_art_crop","generated_original","licensed_landscape"].includes(review.source_kind)) fail("invalid source kind");
    if (entry.fallback !== false || source?.fallback !== false) fail("placeholder fallback remains");
    for (const field of ["source_url","credit","policy_url"]) if (!entry[field] || source?.[field] !== entry[field]) fail(`missing or inconsistent ${field}`);
    try {
      const file = await readFile(path.join(root,"web/public",entry.local_path));
      const hash = createHash("sha256").update(file).digest("hex");
      if (hash !== entry.sha256) fail("manifest hash mismatch");
      if (hash === rejected.get(entry.plane_id)) fail("known text-only asset remains");
      if (hashes.has(hash)) fail("duplicate artwork");
      hashes.add(hash);
      const meta = await sharp(file).metadata();
      await sharp(file).raw().toBuffer();
      if (meta.format !== "webp" || meta.width !== 512 || meta.height !== 512 || file.length > 180*1024) fail("requires decodable 512x512 WebP <=180KiB");
    } catch (error) { fail(`unreadable artwork: ${error.message}`); }
  }
  return { count:manifest.entries.length, errors };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const {count,errors} = await auditPlaneArt();
  if (errors.length) { process.stderr.write(errors.join("\n") + "\n"); process.exitCode=1; }
  else process.stdout.write(`Verified ${count}/85 unique local reviewed text-free Plane artworks; zero fallbacks.\n`);
}
