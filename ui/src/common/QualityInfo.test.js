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
    const info = {
      suffix: 'MP3',
      bitRate: 320,
      rgAlbumGain: -5,
      rgAlbumPeak: 1,
      rgTrackGain: 2.3,
      rgTrackPeak: 0.5,
    }
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
  it('renders album gain info, no peak limit', () => {
    render(
      <QualityInfo
        gainMode="album"
        preAmp={0}
        record={{
          rgAlbumGain: -5,
          rgAlbumPeak: 1,
          rgTrackGain: -2,
          rgTrackPeak: 0.2,
        }}
      />,
    )
    expect(screen.getByText('N/A (-5.00 dB)')).toBeInTheDocument()
  })
  it('renders track gain info, no peak limit capping, preAmp', () => {
    render(
      <QualityInfo
        gainMode="track"
        preAmp={-1}
        record={{
          rgAlbumGain: -5,
          rgAlbumPeak: 1,
          rgTrackGain: 2.3,
          rgTrackPeak: 0.5,
        }}
      />,
    )
    expect(screen.getByText('N/A (1.30 dB)')).toBeInTheDocument()
  })
  it('renders gain info limited by peak', () => {
    render(
      <QualityInfo
        gainMode="track"
        preAmp={-1}
        record={{
          suffix: 'FLAC',
          rgAlbumGain: -5,
          rgAlbumPeak: 1,
          rgTrackGain: 2.3,
          rgTrackPeak: 1,
        }}
      />,
    )
    expect(screen.getByText('FLAC (0.00 dB)')).toBeInTheDocument()
  })
})
