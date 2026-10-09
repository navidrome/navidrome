import { act, render } from '@testing-library/react'
import { Player } from './Player'
import config from '../config'

let playerProps
let state
const dispatch = vi.fn()
vi.mock('navidrome-music-player', () => ({
  default: (props) => {
    playerProps = props
    return null
  },
}))
vi.mock('react-redux', () => ({
  useDispatch: () => dispatch,
  useSelector: (select) => select(state),
}))
vi.mock('react-admin', async () => ({
  ...(await vi.importActual('react-admin')),
  useAuthState: () => ({ authenticated: true }),
  useDataProvider: () => ({}),
  useTranslate: () => (text) => text,
}))
vi.mock('../themes/useCurrentTheme', () => ({ default: () => ({}) }))
vi.mock('../common', () => ({ useInterval: () => {} }))
vi.mock('./styles', () => ({ default: () => ({}) }))
vi.mock('./PlayerToolbar', () => ({ default: () => null }))
vi.mock('./AudioTitle', () => ({ default: () => null }))
vi.mock('react-hotkeys', () => ({ GlobalHotKeys: () => null }))
vi.mock('../subsonic', () => ({
  default: { reportPlayback: vi.fn().mockResolvedValue({}) },
}))
vi.mock('../transcode', () => ({
  detectBrowserProfile: () => ({}),
  decisionService: {
    setProfile: vi.fn(),
    resolveStreamUrl: vi.fn().mockResolvedValue('/stream'),
    prefetchDecisions: vi.fn(),
  },
}))

beforeEach(() => {
  state = {
    player: { queue: [], current: {}, volume: 1 },
    settings: {},
    replayGain: {},
  }
})
afterEach(() => {
  config.instanceName = 'Navidrome'
  document.title = ''
})

it.each(['Navidrome', 'mp3-player'])(
  'uses %s in idle, playing and finished tab titles',
  async (name) => {
    config.instanceName = name
    const { rerender } = render(<Player />)
    expect(document.title).toBe(name)
    state.player.queue = [{ trackId: 'track' }]
    await act(async () => {
      rerender(<Player />)
    })
    await act(async () => {
      playerProps.onAudioPlay({
        trackId: 'track',
        duration: 60,
        currentTime: 0,
        song: { title: 'Moon', artist: 'Artist' },
      })
    })
    expect(document.title).toBe(`Moon - Artist - ${name}`)
    act(() => {
      playerProps.onAudioProgress({ ended: true })
    })
    expect(document.title).toBe(name)
  },
)
