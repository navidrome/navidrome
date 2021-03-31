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
})
