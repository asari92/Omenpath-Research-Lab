import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AIWorklogPage, WorklogMarkdown } from "./AIWorklogPage";

describe("AI Worklog", () => {
  it("renders the repository worklog as semantic Markdown with a generated TOC", () => {
    const { container } = render(<AIWorklogPage />);
    expect(
      screen.getByRole("heading", { name: "AI Worklog — Current", level: 1 }),
    ).toBeVisible();
    expect(
      screen.getByRole("navigation", { name: "On this page" }),
    ).toBeVisible();
    expect(container.querySelector("ul li")).not.toBeNull();
    expect(container.querySelector("pre code")).not.toBeNull();
    expect(container.querySelector("table")).not.toBeNull();
    expect(container.querySelector("[data-table-scroll] table")).not.toBeNull();
  });

  it("exposes the required process evidence from the single source", () => {
    render(<AIWorklogPage />);
    const article = screen.getByRole("article");
    for (const meaning of [
      /AI tools/i,
      /токен/i,
      /время разработки/i,
      /Stage 20/i,
      /Мои ключевые решения/i,
      /Где AI ошибался/i,
      /ручные правки/i,
      /verification/i,
      /будущие/i,
    ]) {
      expect(within(article).getAllByText(meaning).length).toBeGreaterThan(0);
    }
  });

  it("does not execute raw HTML from Markdown", () => {
    const { container } = render(
      <WorklogMarkdown
        source={'# Safe\n<script data-danger="yes">boom()</script>'}
      />,
    );
    expect(container.querySelector("script")).toBeNull();
    expect(screen.getByText(/script data-danger/i)).toBeVisible();
  });
});
