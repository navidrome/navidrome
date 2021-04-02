import * as React from 'react'
import { cleanup, render } from '@testing-library/react'
import { QualityInfo } from './QualityInfo'

describe('<QualityInfo />', () => {
  afterEach(cleanup)

  it('only render FLAC', () => {
    const info = { suffix: 'FLAC', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('FLAC')
    expect(format.innerHTML).toEqual('FLAC')
  })
  it('only render WAV', () => {
    const info = { suffix: 'WAV', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('WAV')
    expect(format.innerHTML).toEqual('WAV')
  })
  it('only render DSF', () => {
    const info = { suffix: 'DSF', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('DSF')
    expect(format.innerHTML).toEqual('DSF')
  })
  it('only render ALAC', () => {
    const info = { suffix: 'ALAC', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('ALAC')
    expect(format.innerHTML).toEqual('ALAC')
  })
  it('only render TTA', () => {
    const info = { suffix: 'TTA', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('TTA')
    expect(format.innerHTML).toEqual('TTA')
  })
  it('only render ATRAC', () => {
    const info = { suffix: 'ATRAC', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('ATRAC')
    expect(format.innerHTML).toEqual('ATRAC')
  })
  it('only render SHN', () => {
    const info = { suffix: 'SHN', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('SHN')
    expect(format.innerHTML).toEqual('SHN')
  })
  it('only render OCG 108', () => {
    const info = { suffix: 'OCG', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('OCG 108')
    expect(format.innerHTML).toEqual('OCG 108')
  })
  it('only render MP3 108', () => {
    const info = { suffix: 'MP3', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('MP3 108')
    expect(format.innerHTML).toEqual('MP3 108')
  })
  it('only render AAC 108', () => {
    const info = { suffix: 'AAC', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('AAC 108')
    expect(format.innerHTML).toEqual('AAC 108')
  })
  it('only render OPUS 108', () => {
    const info = { suffix: 'OPUS', bitRate: 108 }
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('OPUS 108')
    expect(format.innerHTML).toEqual('OPUS 108')
  })
  /* it('render nothing', () => {
    const info = {}
    const { getByText } = render(<QualityInfo record={info} />)
    const format = getByText('OCG 108')
    expect(format).toBeNull()
  }) */
})
