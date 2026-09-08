import React from 'react'
import { render, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { describe, it, expect, vi } from 'vitest'
import { ImageUploadOverlay } from './ImageUploadOverlay'
import config from '../config'

vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useTranslate: () => (x) => x,
    useNotify: () => vi.fn(),
    useRefresh: () => vi.fn(),
  }
})

const renderOverlay = (props) =>
  render(
    <TestContext>
      <ImageUploadOverlay entityType="user" entityId="u1" {...props} />
    </TestContext>,
  )

describe('ImageUploadOverlay', () => {
  it('renders nothing when canEdit is false', () => {
    const { container } = renderOverlay({ canEdit: false })
    expect(container).toBeEmptyDOMElement()
  })

  it('renders when canEdit is true even if artwork upload is off', () => {
    config.enableArtworkUpload = false
    renderOverlay({ canEdit: true })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })
})
