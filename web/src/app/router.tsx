import type { RouteObject } from "react-router-dom";

import { AIWorklogPage } from "../pages/AIWorklogPage";
import { DashboardPage } from "../pages/DashboardPage";
import { EventLogPage } from "../pages/EventLogPage";
import { HelpPage } from "../pages/HelpPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { PortalDetailsPage } from "../pages/PortalDetailsPage";
import { AppShell } from "./AppShell";

export const routes: RouteObject[] = [
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "portals/:id", element: <PortalDetailsPage /> },
      { path: "events", element: <EventLogPage /> },
      {
        path: "help",
        element: <HelpPage />,
      },
      { path: "ai-worklog", element: <AIWorklogPage /> },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
];
