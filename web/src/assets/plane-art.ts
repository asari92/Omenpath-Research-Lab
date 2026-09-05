import manifest from "../../../data/plane_images_manifest.json";

export interface PlaneArtEntry {
  plane_id: number;
  plane_name: string;
  local_path: string;
  source_url: string;
  source_object_id: string | null;
  credit: string;
  policy_url: string;
  sha256: string;
  fallback: boolean;
}

const entries = manifest.entries as readonly PlaneArtEntry[];
const byID = new Map(entries.map((entry) => [entry.plane_id, entry]));
const generic: PlaneArtEntry = {
  plane_id: 0,
  plane_name: "Unknown Plane",
  local_path: manifest.generic_fallback_path,
  source_url: "generated:omenpath-plane-fallback",
  source_object_id: null,
  credit: "Omenpath Research Lab generated fallback",
  policy_url: manifest.policy_url,
  sha256: "",
  fallback: true,
};

export function planeArt(planeID: number): PlaneArtEntry {
  return byID.get(planeID) ?? generic;
}

export function planeArtEntries(): readonly PlaneArtEntry[] {
  return entries;
}

export const fanContentNotice = manifest.fan_content_notice;
