import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { SIMPLE_MOBILE_KEY } from './simpleMobile'
import {
  playerSnapshot,
  paintSimpleShell,
  bindSimpleMobilePlayer,
  unbindSimpleMobilePlayer,
  callPlayerToggle,
  callPlayRandom,
  syncSimpleMobilePlayer,
} from './simpleMobileShell'

const mountShell = () => {
  document.body.innerHTML = `
    <div id="nd-static-simple" data-has-track="0" data-playing="0">
      <button type="button" id="nd-static-full">Open full version</button>
      <img id="nd-static-cover" alt="" hidden />
      <div id="nd-static-title"></div>
      <div id="nd-static-artist"></div>
      <p id="nd-static-hint">Nothing playing</p>
      <p id="nd-static-hint-sub">Shuffle to start</p>
      <button type="button" id="nd-static-play" aria-label="Play"></button>
      <button type="button" id="nd-static-shuffle">Play random songs</button>
    </div>
  `
}

describe('simpleMobileShell', () => {
  beforeEach(() => {
    localStorage.clear()
    unbindSimpleMobilePlayer()
    delete window.__ndPlayRandom
    document.body.innerHTML = ''
  })

  afterEach(() => {
    unbindSimpleMobilePlayer()
    delete window.__ndPlayRandom
    document.body.innerHTML = ''
  })

  it('reads now-playing fields from current song', () => {
    expect(
      playerSnapshot(
        {
          current: {
            name: 'Fallback',
            singer: 'Also fallback',
            cover: 'https://example/cover.jpg',
            song: { title: 'Helplessness Blues', artist: 'Fleet Foxes' },
          },
          queue: [{ name: 'Queued' }],
          savedPlayIndex: 0,
        },
        { paused: false },
      ),
    ).toEqual({
      title: 'Helplessness Blues',
      artist: 'Fleet Foxes',
      cover: 'https://example/cover.jpg',
      hasTrack: true,
      playing: true,
    })
  })

  it('falls back to the queued track after a reload (current is not persisted)', () => {
    expect(
      playerSnapshot(
        {
          current: {},
          savedPlayIndex: 1,
          queue: [
            { name: 'First', singer: 'A', cover: '/a.jpg' },
            {
              name: 'Second',
              singer: 'B',
              cover: '/b.jpg',
              song: { title: 'Second', artist: 'B' },
            },
          ],
        },
        { paused: true },
      ),
    ).toMatchObject({
      title: 'Second',
      artist: 'B',
      cover: '/b.jpg',
      hasTrack: true,
      playing: false,
    })
  })

  it('is empty when the queue and current track are missing', () => {
    expect(playerSnapshot({ queue: [], current: {} }, null)).toEqual({
      title: '',
      artist: '',
      cover: '',
      hasTrack: false,
      playing: false,
    })
  })

  it('paints title, artist, cover, and play/pause into the static shell', () => {
    mountShell()
    paintSimpleShell({
      title: 'Helplessness Blues',
      artist: 'Fleet Foxes',
      cover: 'https://example/cover.jpg',
      hasTrack: true,
      playing: true,
      labels: { pause: 'Pause', play: 'Play' },
    })
    const shell = document.getElementById('nd-static-simple')
    expect(shell.dataset.hasTrack).toBe('1')
    expect(shell.dataset.playing).toBe('1')
    expect(document.getElementById('nd-static-title').textContent).toBe(
      'Helplessness Blues',
    )
    expect(document.getElementById('nd-static-artist').textContent).toBe(
      'Fleet Foxes',
    )
    expect(document.getElementById('nd-static-cover').src).toContain(
      'https://example/cover.jpg',
    )
    expect(document.getElementById('nd-static-cover').hidden).toBe(false)
    const play = document.getElementById('nd-static-play')
    expect(play.getAttribute('aria-label')).toBe('Pause')
    expect(play.getAttribute('aria-pressed')).toBe('true')
    expect(play.disabled).toBe(false)

    paintSimpleShell({
      title: 'Helplessness Blues',
      artist: 'Fleet Foxes',
      cover: 'https://example/cover.jpg',
      hasTrack: true,
      playing: false,
      labels: { pause: 'Pause', play: 'Play' },
    })
    expect(shell.dataset.playing).toBe('0')
    expect(play.getAttribute('aria-label')).toBe('Play')
    expect(play.getAttribute('aria-pressed')).toBe('false')
  })

  it('hides cover and disables play in the empty state', () => {
    mountShell()
    paintSimpleShell({
      hasTrack: false,
      playing: false,
      labels: {
        hint: 'Nothing playing',
        shuffleHint: 'Shuffle to start',
        shuffle: 'Play random songs',
      },
    })
    const shell = document.getElementById('nd-static-simple')
    expect(shell.dataset.hasTrack).toBe('0')
    expect(document.getElementById('nd-static-cover').hidden).toBe(true)
    expect(document.getElementById('nd-static-play').disabled).toBe(true)
    expect(document.getElementById('nd-static-hint').textContent).toBe(
      'Nothing playing',
    )
    expect(document.getElementById('nd-static-hint-sub').textContent).toBe(
      'Shuffle to start',
    )
  })

  it('toggles the bound audio instance and reports via callPlayerToggle', () => {
    const audio = {
      paused: true,
      togglePlay: vi.fn(),
      play: vi.fn(),
      pause: vi.fn(),
    }
    bindSimpleMobilePlayer(audio)
    expect(callPlayerToggle()).toBe(true)
    expect(audio.togglePlay).toHaveBeenCalledTimes(1)
    audio.togglePlay = undefined
    audio.paused = true
    expect(callPlayerToggle()).toBe(true)
    expect(audio.play).toHaveBeenCalled()
    audio.paused = false
    expect(callPlayerToggle()).toBe(true)
    expect(audio.pause).toHaveBeenCalled()
  })

  it('callPlayRandom hits the shuffle bridge when present', () => {
    expect(callPlayRandom()).toBe(false)
    window.__ndPlayRandom = vi.fn()
    expect(callPlayRandom()).toBe(true)
    expect(window.__ndPlayRandom).toHaveBeenCalled()
  })

  it('syncs and listens for play/pause only when simple mode is on', () => {
    mountShell()
    const audio = {
      paused: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }
    const off = syncSimpleMobilePlayer({
      playerState: { queue: [], current: {} },
      audio,
    })
    expect(audio.addEventListener).not.toHaveBeenCalled()
    off()

    localStorage.setItem(SIMPLE_MOBILE_KEY, '1')
    const audio2 = {
      paused: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }
    const stop = syncSimpleMobilePlayer({
      playerState: {
        current: { song: { title: 'Go', artist: 'Artist' }, cover: '' },
        queue: [{ name: 'Go' }],
      },
      audio: audio2,
      labels: { pause: 'Pause' },
    })
    expect(window.__ndPlayer).toBeTruthy()
    expect(document.getElementById('nd-static-title').textContent).toBe('Go')
    expect(
      document.getElementById('nd-static-play').getAttribute('aria-label'),
    ).toBe('Pause')
    expect(audio2.addEventListener).toHaveBeenCalledWith(
      'play',
      expect.any(Function),
    )
    stop()
    expect(audio2.removeEventListener).toHaveBeenCalled()
    expect(window.__ndPlayer).toBeUndefined()
  })
})
