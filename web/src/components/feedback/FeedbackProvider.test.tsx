import { act, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { FeedbackProvider, useFeedback } from "./FeedbackProvider";

function Harness() {
  const feedback = useFeedback();
  return (
    <button
      onClick={() => {
        for (let i = 1; i <= 30; i++) feedback.notify(`Command ${i} completed`);
      }}
    >
      Send notifications
    </button>
  );
}

afterEach(() => vi.useRealTimers());

it("expires the latest notification after eight seconds", () => {
  vi.useFakeTimers();
  render(
    <FeedbackProvider>
      <Harness />
    </FeedbackProvider>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Send notifications" }));
  act(() => vi.advanceTimersByTime(7999));
  expect(screen.getByText("Command 30 completed")).toBeInTheDocument();
  act(() => vi.advanceTimersByTime(1));
  expect(screen.queryByText("Command 30 completed")).not.toBeInTheDocument();
});

it("keeps newest feedback visible with a bounded accessible notification region", async () => {
  render(
    <FeedbackProvider>
      <Harness />
    </FeedbackProvider>,
  );
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Send notifications" }));
  const region = screen.getByLabelText("Notifications");
  expect(region).toHaveAttribute("aria-live", "polite");
  expect(region).toHaveAttribute("data-placement", "page-header");
  expect(within(region).getAllByRole("status")).toHaveLength(1);
  expect(within(region).getByRole("status")).toBeVisible();
  expect(within(region).getByRole("status")).toHaveTextContent(
    "Command 30 completed",
  );
  expect(screen.queryByText("Command 1 completed")).not.toBeInTheDocument();
  await userEvent
    .setup()
    .click(screen.getByRole("button", { name: "Dismiss notification" }));
  expect(within(region).queryByRole("status")).not.toBeInTheDocument();
});
