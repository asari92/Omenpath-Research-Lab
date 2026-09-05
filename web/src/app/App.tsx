import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
} from "react-router-dom";

import { SnapshotProvider } from "../state/SnapshotProvider";
import { routes } from "./router";

export interface AppProps {
  initialEntries?: string[];
}

export function App({ initialEntries }: AppProps) {
  const router = initialEntries
    ? createMemoryRouter(routes, { initialEntries })
    : createBrowserRouter(routes);

  return (
    <SnapshotProvider>
      <RouterProvider router={router} />
    </SnapshotProvider>
  );
}
