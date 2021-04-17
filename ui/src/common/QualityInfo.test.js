import * as React from 'react'
import { cleanup, render } from '@testing-library/react'
import { QualityInfo } from './QualityInfo'

describe('<QualityInfo />', () => {
  afterEach(cleanup)

  it('only render suffix for lossless formats', () => {
    const info = { suffix: 'FLAC', bitRate: 1008 }
    const { queryByText } = render(<QualityInfo record={info} />)
    expect(queryByText('FLAC')).not.toBeNull()
  })
  it('only render suffix and bitrate for lossy formats', () => {
    const info = { suffix: 'MP3', bitRate: 320 }
    const { queryByText } = render(<QualityInfo record={info} />)
    expect(queryByText('MP3 320')).not.toBeNull()
  })
  it('renders placeholder if suffix is missing', () => {
    const info = {}
    const { queryByText } = render(<QualityInfo record={info} />)
    expect(queryByText('N/A')).not.toBeNull()
  })
  it('does not break if record is null', () => {
    const { queryByText } = render(<QualityInfo />)
    expect(queryByText('N/A')).not.toBeNull()
  })
})
