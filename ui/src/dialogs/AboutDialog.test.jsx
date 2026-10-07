import * as React from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import { TestContext } from 'ra-test'
import { DataProviderContext } from 'react-admin'
import { describe, afterEach, it, expect, vi } from 'vitest'
import { AboutDialog, LinkToVersion } from './AboutDialog'
import subsonic from '../subsonic/index.js'
import TableBody from '@material-ui/core/TableBody'
import TableRow from '@material-ui/core/TableRow'
import Table from '@material-ui/core/Table'
import TableCell from '@material-ui/core/TableCell'

describe('<AboutDialog />', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('opens before the ping request completes', () => {
    vi.spyOn(subsonic, 'ping').mockReturnValue(new Promise(() => {}))
    const dataProvider = {
      getOne: vi.fn().mockReturnValue(new Promise(() => {})),
    }

    render(
      <DataProviderContext.Provider value={dataProvider}>
        <TestContext enableReducers>
          <AboutDialog open={true} onClose={vi.fn()} />
        </TestContext>
      </DataProviderContext.Provider>,
    )

    expect(screen.getByText('Navidrome Music Server')).toBeTruthy()
    expect(screen.queryByText('ra.notification.new_version')).toBeNull()
  })
})

const Wrapper = ({ version }) => (
  <Table>
    <TableBody>
      <TableRow>
        <TableCell>
          <LinkToVersion version={version} />
        </TableCell>
      </TableRow>
    </TableBody>
  </Table>
)

describe('<LinkToVersion />', () => {
  afterEach(cleanup)

  it('should not render any link for "dev" version', () => {
    const version = 'dev'
    render(<Wrapper version={version} />)
    expect(screen.queryByRole('link')).toBeNull()
  })

  it('should render nothing while the server version is not known yet', () => {
    render(<Wrapper version="" />)
    expect(screen.queryByRole('link')).toBeNull()
    expect(screen.queryByRole('cell').textContent).toBe('')
  })

  it('should render link to GH tag page for full releases', () => {
    const version = '0.40.0 (300a0292)'
    render(<Wrapper version={version} />)

    const link = screen.queryByRole('link')
    expect(link.href).toBe(
      'https://github.com/navidrome/navidrome/releases/tag/v0.40.0',
    )
    expect(link.textContent).toBe('0.40.0')

    const cell = screen.queryByRole('cell')
    expect(cell.textContent).toBe('0.40.0 (300a0292)')
  })

  it('should render link to GH comparison page for snapshot releases', () => {
    const version = '0.40.0-SNAPSHOT (300a0292)'
    render(<Wrapper version={version} />)

    const link = screen.queryByRole('link')
    expect(link.href).toBe(
      'https://github.com/navidrome/navidrome/compare/v0.40.0...300a0292',
    )
    expect(link.textContent).toBe('0.40.0-SNAPSHOT')

    const cell = screen.queryByRole('cell')
    expect(cell.textContent).toBe('0.40.0-SNAPSHOT (300a0292)')
  })
})
