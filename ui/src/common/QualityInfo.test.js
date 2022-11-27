import * as React from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import { QualityInfo } from './QualityInfo'

describe('<QualityInfo />', () => {
  afterEach(cleanup)

  it('only render suffix for lossless formats', () => {
    const info = { suffix: 'FLAC', bitRate: 1008 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC')).toBeInTheDocument()
  })
  it('only render suffix and bitrate for lossy formats', () => {
    const info = { suffix: 'MP3', bitRate: 320 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('MP3 320')).toBeInTheDocument()
  })
  it('renders placeholder if suffix is missing', () => {
    const info = {}
    render(<QualityInfo record={info} />)
    expect(screen.getByText('N/A')).toBeInTheDocument()
  })
  it('does not break if record is null', () => {
    render(<QualityInfo />)
    expect(screen.getByText('N/A')).toBeInTheDocument()
  })
})
