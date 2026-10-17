import { render, screen } from '@testing-library/react'
import { useMediaQuery } from '@material-ui/core'
import { Title } from './Title'
import config from '../config'

vi.mock('@material-ui/core', async () => ({
  ...(await vi.importActual('@material-ui/core')),
  useMediaQuery: vi.fn(),
}))
vi.mock('react-admin', () => ({ useTranslate: () => (text) => text || '' }))

afterEach(() => {
  config.instanceName = 'Navidrome'
})

it.each(['Navidrome', 'mp3-player', '家の音楽 & Friends'])(
  'shows %s with the desktop page title',
  (name) => {
    config.instanceName = name
    useMediaQuery.mockReturnValue(true)
    render(<Title subTitle="Albums" />)
    expect(screen.getByText(`${name} - Albums`)).toBeInTheDocument()
  },
)

it('preserves the compact mobile subtitle and uses the name when it is empty', () => {
  config.instanceName = 'mp3-player'
  useMediaQuery.mockReturnValue(false)
  const { rerender } = render(<Title subTitle="Albums" />)
  expect(screen.getByText('Albums')).toBeInTheDocument()
  rerender(<Title />)
  expect(screen.getByText('mp3-player')).toBeInTheDocument()
})

it('renders an HTML-like name as text', () => {
  config.instanceName = '<img src=x onerror=alert(1)>'
  useMediaQuery.mockReturnValue(true)
  const { container } = render(<Title />)
  expect(container).toHaveTextContent(config.instanceName)
  expect(container.querySelector('img')).toBeNull()
})
