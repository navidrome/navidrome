import * as React from 'react'
import { cleanup, render } from '@testing-library/react'
import { QuickFilter } from './QuickFilter'
import StarIcon from '@material-ui/icons/Star'

describe('QuickFilter', () => {
  afterEach(cleanup)

  it('renders label if provided', () => {
    const { getByText } = render(
      <QuickFilter resource={'song'} source={'name'} label={'MyLabel'} />
    )
    expect(getByText('MyLabel')).not.toBeNull()
  })

  it('renders resource translation if label is not provided', () => {
    const { getByText } = render(
      <QuickFilter resource={'song'} source={'name'} />
    )
    expect(getByText('resources.song.fields.name')).not.toBeNull()
  })

  it('renders a component label', () => {
    const { getByTestId } = render(
      <QuickFilter
        resource={'song'}
        source={'name'}
        label={<StarIcon data-testid="label-icon-test" />}
      />
    )
    expect(getByTestId('label-icon-test')).not.toBeNull()
  })
})
