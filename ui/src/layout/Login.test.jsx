import { render, screen } from '@testing-library/react'
import Login from './Login'
import config from '../config'

const dispatch = vi.fn()
vi.mock('react-redux', () => ({ useDispatch: () => dispatch }))
vi.mock('react-admin', async () => ({
  ...(await vi.importActual('react-admin')),
  useLogin: () => vi.fn(),
  useNotify: () => vi.fn(),
  useTranslate: () => (text) => text,
  useVersion: () => 1,
}))
vi.mock('../themes/useCurrentTheme', () => ({ default: () => ({}) }))
vi.mock('./Notification', () => ({ default: () => null }))

afterEach(() => {
  config.instanceName = 'Navidrome'
})

it('keeps the default project name and link', () => {
  render(<Login />)
  expect(screen.getAllByText('Navidrome')).toHaveLength(1)
  expect(screen.getByRole('link', { name: 'Navidrome' })).toHaveAttribute(
    'href',
    'https://www.navidrome.org',
  )
})

it('shows a custom name while retaining project attribution', () => {
  config.instanceName = 'mp3-player'
  render(<Login />)
  expect(screen.getByText('mp3-player')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Navidrome' })).toBeInTheDocument()
})
