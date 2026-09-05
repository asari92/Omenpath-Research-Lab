import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <section>
      <h1>Not Found</h1>
      <p>The requested laboratory view does not exist.</p>
      <Link to="/">Return to Dashboard</Link>
    </section>
  );
}
