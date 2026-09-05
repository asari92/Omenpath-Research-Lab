import { describe, expect, it } from "vitest";

import { planeDTO } from "../../test/builders";
import { filterPlanes } from "./plane-filter";

const planes = [
  { ...planeDTO(1, "Ravnica"), aliases: ["City of Guilds"], explored: true },
  { ...planeDTO(2, "Amonkhet"), aliases: ["Naktamun"] },
  {
    ...planeDTO(3, "Zendikar"),
    observers_in_plane: 2,
    observers_waiting_return: 1,
  },
];

describe("filterPlanes", () => {
  it("matches name and aliases case-insensitively", () => {
    expect(filterPlanes(planes, "RAV", "ALL").map((plane) => plane.id)).toEqual(
      [1],
    );
    expect(
      filterPlanes(planes, "naktamun", "ALL").map((plane) => plane.id),
    ).toEqual([2]);
  });

  it("supports the four mutually exclusive views without reordering", () => {
    expect(filterPlanes(planes, "", "ALL").map((plane) => plane.id)).toEqual([
      1, 2, 3,
    ]);
    expect(
      filterPlanes(planes, "", "EXPLORED").map((plane) => plane.id),
    ).toEqual([1]);
    expect(
      filterPlanes(planes, "", "UNEXPLORED").map((plane) => plane.id),
    ).toEqual([2, 3]);
    expect(
      filterPlanes(planes, "", "OBSERVER_PRESENT").map((plane) => plane.id),
    ).toEqual([3]);
  });
});
