import React from 'react'
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LyricsPanel from './LyricsPanel'
import { KaraokeLineRow } from './LyricsLineRows'
import {
  KARAOKE_ANIMATION_MS,
  KARAOKE_CHARACTER_LIFT_PX,
  KARAOKE_LINE_LIFT_PX,
  KARAOKE_MANUAL_SCROLL_PAUSE_MS,
  KARAOKE_TRANSLATION_OPACITY,
} from './lyricsKaraokeConstants'

const theme = createTheme({
  palette: {
    primary: { main: '#2266aa' },
    text: { primary: '#111111', secondary: '#778899' },
  },
})

const panel = (props, selectedTheme = theme) => (
  <ThemeProvider theme={selectedTheme}>
    <LyricsPanel visible {...props} />
  </ThemeProvider>
)

const renderPanel = (props, selectedTheme = theme) => {
  const view = render(panel(props, selectedTheme))
  return {
    ...view,
    rerenderPanel: (nextProps) =>
      view.rerender(panel(nextProps, selectedTheme)),
  }
}

const mainLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'Main line' }],
}

const createTokenizedLyric = (value, first, second) => ({
  synced: true,
  line: [{ start: 0, end: 1000, value }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value,
      cue: [
        { start: 0, end: 500, value: first, byteStart: 0, byteEnd: 3 },
        { start: 500, end: 1000, value: second, byteStart: 5, byteEnd: 8 },
      ],
    },
  ],
})

const tokenizedMainLyric = createTokenizedLyric('Main line', 'Main', 'line')
const tokenizedPronunciationLyric = createTokenizedLyric(
  'mein lain',
  'mein',
  'lain',
)

const voiceCue = (value, agentId, start, end) => ({
  index: 0,
  start,
  end,
  value,
  agentId,
  cue: [{ start, end, value }],
})

const multiAgentLyric = {
  synced: true,
  agents: [
    { id: 'lead', role: 'main' },
    { id: 'all', role: 'group' },
    { id: 'echo', role: 'bg' },
  ],
  line: [{ start: 1000, end: 4000, value: 'Lead all echo' }],
  cueLine: [
    voiceCue('Lead', 'lead', 1000, 2000),
    voiceCue('all', 'all', 1500, 2600),
    voiceCue('echo', 'echo', 2200, 3400),
  ],
}

const createAudio = ({ currentTime = 0, duration = 10 } = {}) => {
  const audio = new EventTarget()
  audio.currentTime = currentTime
  audio.duration = duration
  audio.paused = true
  audio.playbackRate = 1
  audio.seeking = false
  return audio
}

describe('<LyricsPanel />', () => {
  const originalMatchMedia = window.matchMedia

  beforeEach(() => {
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation(() => 0)
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    window.matchMedia = originalMatchMedia
    cleanup()
  })

  it('renders main, stacked pronunciation, and translation in layer order', () => {
    renderPanel({
      mainLyric: tokenizedMainLyric,
      pronunciationLyric: tokenizedPronunciationLyric,
      translationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'translation line' }],
      },
      showPronunciation: true,
      showTranslation: true,
    })

    const pronunciation = screen.getAllByTestId('lyrics-pronunciation-token')
    expect(pronunciation).toHaveLength(2)
    expect(pronunciation[0]).toHaveTextContent('mein')
    expect(pronunciation[1]).toHaveTextContent('lain')
    expect(screen.getByText('translation line')).toBeInTheDocument()
  })

  it('updates every pronunciation-only token with a distinct registry key', () => {
    const audioInstance = createAudio()
    renderPanel({
      mainLyric,
      pronunciationLyric: tokenizedPronunciationLyric,
      showPronunciation: true,
      audioInstance,
    })

    const pronunciation = screen.getAllByTestId('lyrics-pronunciation-token')
    act(() => {
      audioInstance.currentTime = 0.75
      audioInstance.dispatchEvent(new Event('seeking'))
    })

    expect(pronunciation[0]).toHaveAttribute('data-lyrics-state', 'completed')
    expect(pronunciation[1]).toHaveAttribute('data-lyrics-state', 'active')
  })

  it('renders incomplete pronunciation timing as static text', () => {
    renderPanel({
      mainLyric,
      pronunciationLyric: {
        synced: true,
        line: [{ value: 'mein lain' }],
        cueLine: [
          {
            index: 0,
            value: 'mein lain',
            cue: [{ value: 'mein' }, { value: 'lain' }],
          },
        ],
      },
      showPronunciation: true,
    })

    screen.getAllByTestId('lyrics-pronunciation-token').forEach((token) => {
      expect(token).toHaveAttribute('data-timed', 'false')
      expect(token).not.toHaveAttribute('aria-label')
    })
  })

  it('keeps token ref callbacks stable across equivalent rerenders', () => {
    const registerToken = vi.fn()
    const line = {
      start: 0,
      end: 1000,
      value: 'Main',
      tokens: [
        {
          start: 0,
          end: 1000,
          value: 'Main',
          byteStart: 0,
          byteEnd: 3,
        },
      ],
    }
    const row = () => (
      <KaraokeLineRow
        lineIndex={0}
        line={line}
        style={{ color: '#111111' }}
        registerToken={registerToken}
        renderCharacterWave={false}
      />
    )
    const { rerender } = render(row())

    expect(registerToken).toHaveBeenCalledWith(
      expect.any(String),
      expect.any(Object),
      expect.any(HTMLElement),
    )
    registerToken.mockClear()

    rerender(row())

    expect(registerToken).not.toHaveBeenCalled()
  })

  it('renders each translation line under only its closest main line', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 0, end: 1000, value: 'First main line' },
          { start: 1000, end: 2000, value: 'Closest main line' },
          { start: 2000, end: 3000, value: 'Later main line' },
        ],
      },
      translationLyric: {
        synced: true,
        line: [{ start: 1100, end: 2800, value: 'One translated line' }],
      },
      showTranslation: true,
    })

    const translations = screen.getAllByText('One translated line')
    expect(translations).toHaveLength(1)
    expect(
      translations[0].closest('[data-testid="lyrics-line-group"]'),
    ).toHaveTextContent('Closest main line')
  })

  it('suppresses duplicate translations without shifting untimed indexes', () => {
    renderPanel({
      mainLyric: {
        synced: false,
        line: [{ value: 'Hello' }, { value: 'World' }],
      },
      translationLyric: {
        synced: false,
        line: [{ value: 'Hello' }, { value: 'Mundo' }],
      },
      showTranslation: true,
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    expect(groups[0]).toHaveTextContent('Hello')
    expect(groups[0]).not.toHaveTextContent('Mundo')
    expect(groups[1]).toHaveTextContent('World')
    expect(groups[1]).toHaveTextContent('Mundo')
  })

  it('renders line-level pronunciation without inventing word timing', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: '我总要给一些别的' }],
      },
      pronunciationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'wo zong yao gei yi xie bie de' }],
      },
      showPronunciation: true,
      audioInstance: { currentTime: 0.2, paused: true },
    })

    expect(screen.getByText('我总要给一些别的')).toBeInTheDocument()
    const pronunciation = screen.getByText('wo zong yao gei yi xie bie de')
    const group = pronunciation.closest('[data-testid="lyrics-line-group"]')
    expect(group).toHaveAttribute('data-active', 'true')
    expect(
      group.style.getPropertyValue('--lyrics-pronunciation-active-color'),
    ).not.toBe('')
    expect(pronunciation.style.color).toBe('')
  })

  it('uses one shared opacity animation for every static line layer', () => {
    renderPanel({
      mainLyric,
      pronunciationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'main pronunciation' }],
      },
      translationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'translated line' }],
      },
      showPronunciation: true,
      showTranslation: true,
      audioInstance: { currentTime: 0.5, paused: true },
    })

    const mainRow = screen
      .getByTestId('lyrics-line-group')
      .querySelector('[data-tokenized]')
    const translationRow = screen
      .getByText('translated line')
      .closest('[data-tokenized]')
    const pronunciation = screen.getAllByTestId('lyrics-pronunciation-token')

    ;[mainRow, translationRow].forEach((row) => {
      expect(row).toHaveAttribute('data-layer-animation', 'shared-opacity')
      expect(row).toHaveAttribute('data-tokenized', 'false')
      expect(row.style.opacity).toBe('')
      expect(window.getComputedStyle(row).transition).toContain(
        `opacity ${KARAOKE_ANIMATION_MS}ms`,
      )
    })
    pronunciation.forEach((token) =>
      expect(token).toHaveAttribute('data-timed', 'false'),
    )
    expect(window.getComputedStyle(translationRow).opacity).toBe(
      String(KARAOKE_TRANSLATION_OPACITY),
    )
  })

  it('raises a line once and keeps it elevated after release', () => {
    const { rerenderPanel } = renderPanel({
      mainLyric,
      audioInstance: { currentTime: 0.5, paused: true },
    })

    const group = screen.getByTestId('lyrics-line-group')
    const activeStyle = window.getComputedStyle(group)
    expect(group).toHaveAttribute('data-highlight-active', 'true')
    expect(group).toHaveAttribute('data-raised', 'true')
    expect(group).toHaveAttribute('data-line-motion', 'line')
    expect(activeStyle.transform).toBe(`translateY(-${KARAOKE_LINE_LIFT_PX}px)`)

    rerenderPanel({
      mainLyric,
      audioInstance: { currentTime: 1.1, paused: true },
    })

    const releasedStyle = window.getComputedStyle(group)
    expect(group).toHaveAttribute('data-highlight-active', 'false')
    expect(group).toHaveAttribute('data-raised', 'true')
    expect(releasedStyle.transform).toBe(
      `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
    )
  })

  it('keeps every word-timed layer on the same rise and release lifecycle', () => {
    const offsetPronunciationLyric = {
      ...tokenizedPronunciationLyric,
      cueLine: [
        {
          ...tokenizedPronunciationLyric.cueLine[0],
          cue: [
            { start: 100, end: 850, value: 'mein' },
            { start: 850, end: 1000, value: 'lain' },
          ],
        },
      ],
    }
    const propsAt = (currentTime) => ({
      mainLyric: tokenizedMainLyric,
      pronunciationLyric: offsetPronunciationLyric,
      translationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'translated line' }],
      },
      showPronunciation: true,
      showTranslation: true,
      audioInstance: { currentTime, paused: true },
    })
    const { rerenderPanel } = renderPanel(propsAt(0.25))

    const group = screen.getByTestId('lyrics-line-group')
    const translation = screen
      .getByText('translated line')
      .closest('[data-tokenized]')
    const mainToken = screen.getAllByTestId('lyrics-token')[0]
    const pronunciationToken = screen.getAllByTestId(
      'lyrics-pronunciation-token',
    )[0]
    const mainCharacters = mainToken.querySelectorAll(
      '[data-lyrics-character="true"]',
    )
    const pronunciationCharacters = pronunciationToken.querySelectorAll(
      '[data-lyrics-character="true"]',
    )

    expect(group).toHaveAttribute('data-line-motion', 'character')
    expect(group).toHaveAttribute('data-character-wave', 'true')
    expect(window.getComputedStyle(group).transform).toBe('translateY(0)')
    expect(window.getComputedStyle(translation).transform).toBe(
      `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
    )
    expect(mainCharacters).toHaveLength(4)
    expect(pronunciationCharacters).toHaveLength(4)
    expect(mainCharacters[0].style.transform).toBe(
      `translateY(-${KARAOKE_CHARACTER_LIFT_PX.toFixed(4)}px)`,
    )
    expect(mainCharacters[3].style.transform).not.toBe(
      mainCharacters[0].style.transform,
    )
    expect(pronunciationCharacters[0].style.transform).toBe(
      mainCharacters[0].style.transform,
    )
    expect(mainToken.style.backgroundImage).toBe('none')
    expect(mainCharacters[0].style.backgroundImage).toContain('linear-gradient')

    rerenderPanel(propsAt(1.1))

    expect(group).toHaveAttribute('data-highlight-active', 'false')
    expect(group).toHaveAttribute('data-raised', 'true')
    expect(group).toHaveAttribute('data-character-wave', 'false')
    expect(
      group.querySelectorAll('[data-lyrics-character="true"]'),
    ).toHaveLength(0)
    expect(window.getComputedStyle(group).transform).toBe(
      `translateY(-${KARAOKE_LINE_LIFT_PX}px)`,
    )
    expect(window.getComputedStyle(translation).transform).toBe('')
    const idleText = screen
      .getAllByTestId('lyrics-token')[0]
      .querySelector('[data-lyrics-wave-text="true"]')
    expect(idleText.style.backgroundImage).toContain('linear-gradient')
  })

  it('keeps detailed grapheme markup on active lines without changing token layout', () => {
    const values = ['First word', 'Second word', 'Active word', 'Fourth word']
    const lyricFor = (pronunciation = false) => ({
      synced: true,
      line: values.map((value, index) => ({
        start: index * 1000,
        end: (index + 1) * 1000,
        value: pronunciation ? `spoken ${index}` : value,
      })),
      cueLine: values.map((value, index) => ({
        index,
        start: index * 1000,
        end: (index + 1) * 1000,
        value: pronunciation ? `spoken ${index}` : value,
        cue: (pronunciation ? `spoken ${index}` : value)
          .split(' ')
          .map((word, wordIndex, words) => ({
            start: index * 1000 + (wordIndex * 1000) / words.length,
            end: index * 1000 + ((wordIndex + 1) * 1000) / words.length,
            value: word,
          })),
      })),
    })
    const propsAt = (currentTime) => ({
      mainLyric: lyricFor(),
      pronunciationLyric: lyricFor(true),
      showPronunciation: true,
      audioInstance: { currentTime, paused: true },
    })
    const { rerenderPanel } = renderPanel(propsAt(2.5))

    let groups = screen.getAllByTestId('lyrics-line-group')
    expect(
      groups.map((group) =>
        Boolean(group.querySelector('[data-lyrics-character="true"]')),
      ),
    ).toEqual([false, false, true, false])
    expect(
      groups[2].querySelectorAll('[data-lyrics-wave-measure="true"]'),
    ).not.toHaveLength(0)
    const activeRow = groups[2].querySelector('[data-wrapped]')
    const spacers = activeRow.querySelectorAll(
      ':scope > span:not([data-stacked-token="true"])',
    )
    expect(activeRow).toHaveAttribute('data-wrapped', 'false')
    expect(spacers).not.toHaveLength(0)
    spacers.forEach((spacer) =>
      expect(window.getComputedStyle(spacer).display).toBe('inline'),
    )

    rerenderPanel(propsAt(3.5))
    groups = screen.getAllByTestId('lyrics-line-group')
    expect(
      groups[2].querySelectorAll('[data-lyrics-character="true"]'),
    ).toHaveLength(0)
    expect(
      groups[3].querySelectorAll('[data-lyrics-character="true"]'),
    ).not.toHaveLength(0)
  })

  it('renders unsynced lyrics as static selectable text', () => {
    renderPanel({
      mainLyric: {
        synced: false,
        line: [{ value: 'first plain line' }, { value: 'second plain line' }],
      },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    expect(groups).toHaveLength(2)
    groups.forEach((group) => {
      expect(group).toHaveAttribute('data-active', 'true')
      expect(group).toHaveAttribute('data-lifecycle', 'active')
      expect(group).toHaveAttribute('data-highlight-active', 'true')
      expect(group).not.toHaveAttribute('aria-current')
      expect(group).toHaveAttribute('data-scroll-target', 'false')
    })
  })

  it('tracks overlapping lines while selecting one primary line', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 1000, end: 4000, value: 'Lead vocal' },
          { start: 2000, end: 3000, value: 'Answer vocal' },
          { start: 5000, end: 6000, value: 'Later vocal' },
        ],
      },
      audioInstance: { currentTime: 2.5, paused: true },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    expect(groups[0]).toHaveAttribute('data-active', 'true')
    expect(groups[1]).toHaveAttribute('data-active', 'true')
    expect(groups[2]).toHaveAttribute('data-active', 'false')
    expect(groups[0]).not.toHaveAttribute('aria-current')
    expect(groups[1]).toHaveAttribute('aria-current', 'true')
    expect(groups[1]).toHaveAttribute('data-scroll-target', 'true')
  })

  it('keeps multi-agent cue lines in separate voice lanes', () => {
    renderPanel({
      mainLyric: multiAgentLyric,
      audioInstance: { currentTime: 2.5, paused: true },
    })

    const lanes = screen.getAllByTestId('lyrics-voice-lane')
    expect(lanes).toHaveLength(3)
    expect(lanes[0]).toHaveTextContent('Lead')
    expect(lanes[1]).toHaveTextContent('all')
    expect(lanes[1].style.fontStyle).toBe('italic')
    expect(lanes[2]).toHaveTextContent('echo')
    expect(lanes[2].style.fontStyle).toBe('italic')

    const mainToken = lanes[0].querySelector('[data-testid="lyrics-token"]')
    const emphasisToken = lanes[1].querySelector('[data-testid="lyrics-token"]')
    const emphasisCharacter = emphasisToken.querySelector(
      '[data-lyrics-character="true"]',
    )
    expect(mainToken).toHaveTextContent('Lead')
    expect(emphasisToken.style.paddingInlineEnd).not.toBe('')
    expect(emphasisToken.style.marginInlineEnd).toBe(
      `-${emphasisToken.style.paddingInlineEnd}`,
    )
    expect(emphasisCharacter.style.paddingInlineEnd).toBe(
      emphasisToken.style.paddingInlineEnd,
    )
  })

  it('seeks and synchronizes immediately when a line is activated', () => {
    const audioInstance = { currentTime: 0 }
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 2300, end: 3200, value: 'Seek line' }],
      },
      audioInstance,
    })

    const group = screen
      .getByText('Seek line')
      .closest('[data-testid="lyrics-line-group"]')
    fireEvent.click(group)
    expect(audioInstance.currentTime).toBe(2.3)
    expect(group).toHaveAttribute('data-active', 'true')

    audioInstance.currentTime = 0
    fireEvent.keyDown(group, { key: 'Enter' })
    expect(audioInstance.currentTime).toBe(2.3)
  })

  it('pauses auto-scroll only for genuine manual scroll intent', async () => {
    vi.useFakeTimers()
    const requestAnimationFrameSpy = vi
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation(() => 0)
    renderPanel({
      mainLyric,
      audioInstance: { currentTime: 0.5, paused: true },
    })

    const body = screen.getByTestId('lyrics-scroll-body')
    const initialFrames = requestAnimationFrameSpy.mock.calls.length
    fireEvent.wheel(body)
    expect(body).toHaveAttribute('data-scrollbar-visible', 'true')
    expect(requestAnimationFrameSpy).toHaveBeenCalledTimes(initialFrames)

    act(() => {
      vi.advanceTimersByTime(KARAOKE_MANUAL_SCROLL_PAUSE_MS)
    })
    await waitFor(() => {
      expect(requestAnimationFrameSpy.mock.calls.length).toBeGreaterThan(
        initialFrames,
      )
    })
  })

  it('resets scroll position when lyric content changes', () => {
    const { rerenderPanel } = renderPanel({ mainLyric })
    const body = screen.getByTestId('lyrics-scroll-body')
    body.scrollTop = 180

    rerenderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'Different song' }],
      },
    })
    expect(body.scrollTop).toBe(0)
  })

  it('respects reduced motion and empty states', async () => {
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }))
    const { unmount } = renderPanel({ mainLyric })
    await waitFor(() => {
      expect(screen.getByTestId('lyrics-scroll-body')).toHaveAttribute(
        'data-reduced-motion',
        'true',
      )
    })
    unmount()

    renderPanel({ mainLyric: null, loading: true })
    expect(screen.getByTestId('lyrics-empty-state')).toHaveTextContent(
      'Loading lyrics',
    )
  })

  it('dims translations relative to their semantic theme color', () => {
    const sameTextTheme = createTheme({
      palette: {
        primary: { main: '#ffffff' },
        text: { primary: '#ffffff', secondary: '#ffffff' },
      },
    })
    renderPanel(
      {
        mainLyric: {
          synced: true,
          line: [
            { start: 0, end: 1000, value: 'Active' },
            { start: 1000, end: 2000, value: 'Future' },
          ],
        },
        translationLyric: {
          synced: true,
          line: [
            { start: 0, end: 1000, value: 'Activa' },
            { start: 1000, end: 2000, value: 'Futura' },
          ],
        },
        showTranslation: true,
        audioInstance: { currentTime: 0.5, paused: true },
      },
      sameTextTheme,
    )

    const groups = screen.getAllByTestId('lyrics-line-group')
    const activeTranslation = screen
      .getByText('Activa')
      .closest('[data-tokenized]')
    const futureTranslation = screen
      .getByText('Futura')
      .closest('[data-tokenized]')
    groups.forEach((group) => {
      expect(
        group.style.getPropertyValue('--lyrics-translation-active-color'),
      ).toBe('#ffffff')
    })
    expect(groups[0]).toHaveAttribute('data-active', 'true')
    expect(groups[1]).toHaveAttribute('data-active', 'false')
    const layerOpacity = (group) =>
      window
        .getComputedStyle(group)
        .getPropertyValue('--lyrics-layer-opacity')
        .trim()
    expect(groups.map(layerOpacity)).toEqual(['1', '0.49'])
    expect(window.getComputedStyle(activeTranslation).opacity).toBe(
      String(KARAOKE_TRANSLATION_OPACITY),
    )
    expect(Number(window.getComputedStyle(futureTranslation).opacity)).toBe(
      0.49 * KARAOKE_TRANSLATION_OPACITY,
    )
    ;[activeTranslation, futureTranslation].forEach((translation) =>
      expect(window.getComputedStyle(translation).transition).toContain(
        `opacity ${KARAOKE_ANIMATION_MS}ms`,
      ),
    )
  })
})
