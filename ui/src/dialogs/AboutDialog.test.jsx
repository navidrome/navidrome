import * as React from 'react'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { AboutDialog, LinkToVersion } from './AboutDialog'
import subsonic from '../subsonic'
import TableBody from '@material-ui/core/TableBody'
import TableRow from '@material-ui/core/TableRow'
import Table from '@material-ui/core/Table'
import TableCell from '@material-ui/core/TableCell'

vi.mock('../subsonic', () => ({
  default: { ping: vi.fn() },
}))

vi.mock('react-admin', async (importOriginal) => ({
  ...(await importOriginal()),
  useGetOne: () => ({ data: undefined, loading: true }),
  usePermissions: () => ({ permissions: 'user' }),
  useTranslate: () => (key) => key,
  useNotify: () => vi.fn(),
}))

describe('<AboutDialog />', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('opens before the server has reported its version', () => {
    subsonic.ping.mockReturnValue(new Promise(() => {}))
    render(<AboutDialog open onClose={() => {}} />)

    expect(screen.getByText('menu.version:')).toBeInTheDocument()
    expect(screen.queryByText('ra.notification.new_version')).toBeNull()
  })

  it('stays usable when the server cannot be reached', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    subsonic.ping.mockRejectedValue(new Error('offline'))
    render(<AboutDialog open onClose={() => {}} />)

    await waitFor(() =>
      expect(consoleError).toHaveBeenCalledWith(
        'error pinging server',
        expect.any(Error),
      ),
    )
    expect(screen.getByText('navidrome.org')).toBeInTheDocument()
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

  it('should render an empty version as empty text', () => {
    render(<Wrapper version="" />)
    expect(screen.queryByRole('link')).toBeNull()
    expect(screen.getByRole('cell').textContent).toBe('')
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
