import type { PlaneDTO } from "../../api/types";

export type PlaneFilter =
  "ALL" | "UNEXPLORED" | "EXPLORED" | "OBSERVER_PRESENT";

export function filterPlanes(
  planes: readonly PlaneDTO[],
  query: string,
  filter: PlaneFilter,
): readonly PlaneDTO[] {
  const needle = query.trim().toLocaleLowerCase();
  return planes.filter((plane) => {
    const matchesQuery =
      needle === "" ||
      plane.name.toLocaleLowerCase().includes(needle) ||
      plane.aliases.some((alias) => alias.toLocaleLowerCase().includes(needle));
    if (!matchesQuery) return false;
    switch (filter) {
      case "ALL":
        return true;
      case "UNEXPLORED":
        return !plane.explored;
      case "EXPLORED":
        return plane.explored;
      case "OBSERVER_PRESENT":
        return plane.observers_in_plane > 0;
    }
  });
}
