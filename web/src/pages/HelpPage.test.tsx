import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { expect, it } from "vitest";
import { routes } from "../app/router";

it("provides every field-guide chapter and concrete mechanics", () => {
  const help = routes[0]!.children!.find((route) => route.path === "help")!;
  render(<RouterProvider router={createMemoryRouter([help], { initialEntries: ["/help"] })} />);
  for (const name of ["The Multiverse", "Laboratory interface", "Two kinds of energy", "Portal lifecycle", "Risk and instability", "Observers and transit", "Research and creatures", "Portal commands", "Extraction", "Leyline Override", "Recommendations", "Tutorial recap", "Glossary"]) {
    expect(screen.getByRole("heading", { name, exact: true })).toBeInTheDocument();
  }
  for (const copy of [/85 Planes/, /20 permanent Observers/, /5–15 seconds/, /20 seconds/, /WAITING_RETURN/, /30 Lab Energy/, /5 seconds of synchronization/, /longest-waiting/, /only the first/i]) {
    expect(document.body).toHaveTextContent(copy);
  }
});
