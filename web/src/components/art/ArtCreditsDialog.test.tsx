import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ArtCreditsDialog } from './ArtCreditsDialog';

describe('ArtCreditsDialog', () => {
  it('renders safe source, credit and policy links', () => {
    render(<ArtCreditsDialog open onClose={vi.fn()} />);

    const source = screen.getAllByRole('link', { name: /source/i })[0];
    expect(source).toHaveAttribute('target', '_blank');
    expect(source).toHaveAttribute('rel', 'noreferrer');
    expect(screen.getByText(/unofficial fan content/i)).toBeInTheDocument();
  });
});
