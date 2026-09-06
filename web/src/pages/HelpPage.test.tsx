import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { expect, it } from "vitest";
import { routes } from "../app/router";

it("provides every field-guide chapter and concrete mechanics", () => {
  const help = routes[0]!.children!.find((route) => route.path === "help")!;
  render(
    <RouterProvider
      router={createMemoryRouter([help], { initialEntries: ["/help"] })}
    />,
  );
  for (const name of [
    "The Multiverse",
    "Laboratory interface",
    "Two kinds of energy",
    "Portal lifecycle",
    "Risk and instability",
    "Observers and transit",
    "Research and creatures",
    "Portal commands",
    "Extraction",
    "Leyline Override",
    "Recommendations",
    "Tutorial recap",
    "Glossary",
  ]) {
    expect(screen.getByRole("heading", { name })).toBeInTheDocument();
  }
  for (const copy of [
    /85 Planes/,
    /20 permanent Observers/,
    /5–15 seconds/,
    /20 seconds/,
    /WAITING_RETURN/,
    /30 Lab Energy/,
    /5 seconds of synchronization/,
    /longest-waiting/,
    /only the first/i,
  ]) {
    expect(document.body).toHaveTextContent(copy);
  }
  const recap = screen.getByRole("heading", {
    name: "Tutorial recap",
  }).parentElement;
  expect(recap).toHaveTextContent(
    /Step 0.*introduction.*Step 1.*select.*highlighted Portal.*Dashboard.*Step 2.*creatures.*Step 3.*SEND.*OUTBOUND.*Step 4.*STABILIZE.*Step 5.*CRITICAL.*Step 6.*RECALL.*RETURNING.*Step 7.*Wait.*AVAILABLE.*EXPLORED.*Step 8.*Event Log.*Step 9.*Live/i,
  );
  expect(recap).not.toHaveTextContent(/Open.*Details|Details button/i);
  expect(recap?.textContent?.match(/Step \d+/g)).toEqual(
    Array.from({ length: 10 }, (_, index) => `Step ${index}`),
  );
});
