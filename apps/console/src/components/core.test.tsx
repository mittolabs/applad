import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { statusVariant, StatusChip } from './status-chip';
import { totalPages } from './search-list';
import { resolveTheme } from '@/stores/theme';

describe('statusVariant', () => {
  it('maps known status words to variants', () => {
    expect(statusVariant('active')).toBe('success');
    expect(statusVariant('Deployed')).toBe('success');
    expect(statusVariant('pending')).toBe('warning');
    expect(statusVariant('failed')).toBe('danger');
    expect(statusVariant('running')).toBe('info');
  });
  it('falls back to neutral for unknown words', () => {
    expect(statusVariant('bananas')).toBe('neutral');
  });
});

describe('StatusChip', () => {
  it('renders the label with underscores humanized', () => {
    render(<StatusChip label="pending_review" />);
    expect(screen.getByText('pending review')).toBeInTheDocument();
  });
});

describe('totalPages', () => {
  it('computes page count and clamps to >= 1', () => {
    expect(totalPages(0, 12)).toBe(1);
    expect(totalPages(12, 12)).toBe(1);
    expect(totalPages(13, 12)).toBe(2);
    expect(totalPages(25, 6)).toBe(5);
  });
});

describe('resolveTheme', () => {
  it('resolves explicit modes directly', () => {
    expect(resolveTheme('light')).toBe('light');
    expect(resolveTheme('dark')).toBe('dark');
  });
  it('resolves system via matchMedia (stubbed to false → dark)', () => {
    expect(resolveTheme('system')).toBe('dark');
  });
});
