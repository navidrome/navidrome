import React from 'react'
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { List } from './List'

// Only stub the heavy react-admin List controller (data fetching, router sync);
// everything else, including our own Pagination/perPageStore wiring, stays real
// so a bad import (the bug this test guards against) throws on render.
vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    List: ({ children }) => <div data-testid="ra-list">{children}</div>,
  }
})

describe('List', () => {
  it('renders without throwing and shows its children', () => {
    render(
      <List resource="song">
        <div>list content</div>
      </List>,
    )
    expect(screen.getByTestId('ra-list')).toBeInTheDocument()
    expect(screen.getByText('list content')).toBeInTheDocument()
  })
})
