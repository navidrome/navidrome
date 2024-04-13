import * as React from 'react'
import { render, cleanup, screen } from '@testing-library/react'
import { MultiLineTextField } from './MultiLineTextField'

describe('<MultiLineTextField />', () => {
  afterEach(cleanup)

  it('should render each line in a separated div', () => {
    const record = { comment: 'line1\nline2' }
    render(<MultiLineTextField record={record} source={'comment'} />)
    expect(screen.queryByTestId('comment.0').textContent).toBe('line1')
    expect(screen.queryByTestId('comment.1').textContent).toBe('line2')
  })

  it.each([null, undefined])(
    'should render the emptyText when value is %s',
    (body) => {
      render(
        <MultiLineTextField
          record={{ id: 123, body }}
          emptyText="NA"
          source="body"
        />,
      )
      expect(screen.getByText('NA')).toBeInTheDocument()
    },
  )
})
