import {
  createBrowserRouter,
  createMemoryRouter,
  RouterProvider,
} from "react-router-dom";

import { routes } from "./router";

export interface AppProps {
  initialEntries?: string[];
}

export function App({ initialEntries }: AppProps) {
  const router = initialEntries
    ? createMemoryRouter(routes, { initialEntries })
    : createBrowserRouter(routes);

  return <RouterProvider router={router} />;
}
