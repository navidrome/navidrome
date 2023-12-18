import * as React from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import { QuickFilter } from './QuickFilter'
import StarIcon from '@material-ui/icons/Star'

describe('QuickFilter', () => {
  afterEach(cleanup)

  it('renders label if provided', () => {
    render(<QuickFilter resource={'song'} source={'name'} label={'MyLabel'} />)
    expect(screen.getByText('MyLabel')).not.toBeNull()
  })

  it('renders resource translation if label is not provided', () => {
    render(<QuickFilter resource={'song'} source={'name'} />)
    expect(screen.getByText('resources.song.fields.name')).not.toBeNull()
  })

  it('renders a component label', () => {
    render(
      <QuickFilter
        resource={'song'}
        source={'name'}
        label={<StarIcon data-testid="label-icon-test" />}
      />,
    )
    expect(screen.getByTestId('label-icon-test')).not.toBeNull()
  })
})
