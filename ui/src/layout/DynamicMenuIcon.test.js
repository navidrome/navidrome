import * as React from 'react'
import { render } from '@testing-library/react'
import { createMemoryHistory } from 'history'
import { Router } from 'react-router-dom'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import DynamicMenuIcon from './DynamicMenuIcon'

describe('<DynamicMenuIcon />', () => {
  it('renders icon if no activeIcon is specified', () => {
    const history = createMemoryHistory()
    const route = '/test'
    history.push(route)

    const { getByTestId } = render(
      <Router history={history}>
        <DynamicMenuIcon icon={StarIcon} path={'test'} />
      </Router>
    )
    expect(getByTestId('icon')).not.toBeNull()
  })

  it('renders icon if path does not match the URL', () => {
    const history = createMemoryHistory()
    const route = '/path'
    history.push(route)

    const { getByTestId } = render(
      <Router history={history}>
        <DynamicMenuIcon
          icon={StarIcon}
          activeIcon={StarBorderIcon}
          path={'otherpath'}
        />
      </Router>
    )
    expect(getByTestId('icon')).not.toBeNull()
  })

  it('renders activeIcon if path matches the URL', () => {
    const history = createMemoryHistory()
    const route = '/path'
    history.push(route)

    const { getByTestId } = render(
      <Router history={history}>
        <DynamicMenuIcon
          icon={StarIcon}
          activeIcon={StarBorderIcon}
          path={'path'}
        />
      </Router>
    )
    expect(getByTestId('activeIcon')).not.toBeNull()
  })
})
