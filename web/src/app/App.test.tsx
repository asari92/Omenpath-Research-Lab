import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { App } from './App';

describe('application routes', () => {
  it('renders the shared shell and dashboard at /', async () => {
    render(<App initialEntries={['/']} />);

    expect(await screen.findByRole('banner')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: /laboratory overview/i }),
    ).toBeInTheDocument();
  });

  it.each([
    ['/events', /event log/i],
    ['/ai-worklog', /ai worklog/i],
    ['/unknown', /not found/i],
  ])('maps %s to its page boundary', async (path, heading) => {
    render(<App initialEntries={[path]} />);

    expect(await screen.findByRole('heading', { name: heading })).toBeInTheDocument();
  });
});
