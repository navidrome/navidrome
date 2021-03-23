import * as React from 'react'
import { cleanup, render } from '@testing-library/react'
import { LinkToVersion } from './AboutDialog'
import TableBody from '@material-ui/core/TableBody'
import TableRow from '@material-ui/core/TableRow'
import Table from '@material-ui/core/Table'

const Wrapper = ({ version }) => (
  <Table>
    <TableBody>
      <TableRow>
        <LinkToVersion version={version} />
      </TableRow>
    </TableBody>
  </Table>
)

describe('<LinkToVersion />', () => {
  afterEach(cleanup)

  it('should not render any link for "dev" version', () => {
    const version = 'dev'
    const { queryByRole } = render(<Wrapper version={version} />)
    expect(queryByRole('link')).toBeNull()
  })

  it('should render link to GH tag page for full releases', () => {
    const version = '0.40.0 (300a0292)'
    const { queryByRole } = render(<Wrapper version={version} />)

    const link = queryByRole('link')
    expect(link.href).toBe(
      'https://github.com/navidrome/navidrome/releases/tag/v0.40.0'
    )
    expect(link.textContent).toBe('0.40.0')

    const cell = queryByRole('cell')
    expect(cell.textContent).toBe('0.40.0 (300a0292)')
  })

  it('should render link to GH comparison page for snapshot releases', () => {
    const version = '0.40.0-SNAPSHOT (300a0292)'
    const { queryByRole } = render(<Wrapper version={version} />)

    const link = queryByRole('link')
    expect(link.href).toBe(
      'https://github.com/navidrome/navidrome/compare/v0.40.0...300a0292'
    )
    expect(link.textContent).toBe('0.40.0-SNAPSHOT')

    const cell = queryByRole('cell')
    expect(cell.textContent).toBe('0.40.0-SNAPSHOT (300a0292)')
  })
})
