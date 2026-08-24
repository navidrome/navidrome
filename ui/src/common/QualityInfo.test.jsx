import * as React from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import { QualityInfo } from './QualityInfo'

describe('<QualityInfo />', () => {
  afterEach(cleanup)

  it('only renders suffix for lossless formats without stream details', () => {
    const info = { suffix: 'FLAC', bitRate: 1008 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC')).toBeInTheDocument()
  })
  it('renders sample rate and bit depth for lossless formats', () => {
    const info = {
      suffix: 'FLAC',
      bitRate: 1008,
      sampleRate: 96000,
      bitDepth: 24,
    }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC 96/24')).toBeInTheDocument()
  })
  it('renders CD quality without a trailing zero', () => {
    const info = { suffix: 'FLAC', sampleRate: 44100, bitDepth: 16 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC 44.1/16')).toBeInTheDocument()
  })
  it('keeps one decimal for rates that are not whole kHz', () => {
    const info = { suffix: 'FLAC', sampleRate: 88200, bitDepth: 24 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC 88.2/24')).toBeInTheDocument()
  })
  it('renders the sample rate alone when bit depth is unknown', () => {
    const info = { suffix: 'FLAC', sampleRate: 192000 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('FLAC 192')).toBeInTheDocument()
  })
  it('renders stream details for other lossless containers', () => {
    const info = { suffix: 'WAV', sampleRate: 48000, bitDepth: 32 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('WAV 48/32')).toBeInTheDocument()
  })
  it('leaves DSD showing the container alone', () => {
    const info = { suffix: 'DSF', sampleRate: 2822400, bitDepth: 1 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('DSF')).toBeInTheDocument()
  })
  it('ignores stream details for lossy formats', () => {
    const info = { suffix: 'MP3', bitRate: 320, sampleRate: 44100 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('MP3 320')).toBeInTheDocument()
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

  it('shows transcode arrow when transcodeStream is provided', () => {
    const info = { suffix: 'FLAC', bitRate: 1008 }
    const transcodeStream = { codec: 'opus', audioBitrate: 128000 }
    render(<QualityInfo record={info} transcodeStream={transcodeStream} />)
    expect(screen.getByText('FLAC → OPUS 128')).toBeInTheDocument()
  })

  it('shows transcode with lossy source including bitrate', () => {
    const info = { suffix: 'FLAC', bitRate: 1008 }
    const transcodeStream = { codec: 'mp3', audioBitrate: 320000 }
    render(<QualityInfo record={info} transcodeStream={transcodeStream} />)
    expect(screen.getByText('FLAC → MP3 320')).toBeInTheDocument()
  })

  it('does not show arrow when isDirectPlay is true', () => {
    const info = { suffix: 'MP3', bitRate: 320 }
    render(<QualityInfo record={info} isDirectPlay={true} />)
    expect(screen.getByText('MP3 320')).toBeInTheDocument()
  })

  it('behaves normally when no transcode props are passed', () => {
    const info = { suffix: 'MP3', bitRate: 320 }
    render(<QualityInfo record={info} />)
    expect(screen.getByText('MP3 320')).toBeInTheDocument()
  })
})
