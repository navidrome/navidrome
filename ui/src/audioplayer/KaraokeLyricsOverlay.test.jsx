import React from 'react'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import KaraokeLyricsOverlay from './KaraokeLyricsOverlay'

const DEFAULT_LINE_HEIGHT_TEXT = '1.30'
const NEXT_LINE_HEIGHT_TEXT = '1.32'

const audioInstance = {
  currentTime: 0,
  paused: true,
  seeking: false,
  playbackRate: 1,
}

const buildLyric = (kind, lang, value) => ({
  kind,
  lang,
  synced: true,
  line: [{ start: 1000, value }],
})

const renderOverlay = (props = {}) =>
  render(
    <KaraokeLyricsOverlay
      visible={true}
      mainLyric={buildLyric('main', 'ja', 'こんにちは')}
      translationLyric={buildLyric('translation', 'en', 'Hello')}
      pronunciationLyric={buildLyric('pronunciation', 'ja-Latn', 'konnichiwa')}
      showTranslation={false}
      showPronunciation={true}
      translationEnabled={true}
      pronunciationEnabled={true}
      onToggleTranslation={() => {}}
      onTogglePronunciation={() => {}}
      audioInstance={audioInstance}
      onClose={() => {}}
      {...props}
    />,
  )

describe('<KaraokeLyricsOverlay /> behavior', () => {
  beforeEach(() => {
    localStorage.clear()
    window.innerWidth = 1200
    window.innerHeight = 900
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation(() => 1)
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
  })

  afterEach(() => {
    vi.restoreAllMocks()
    cleanup()
  })

  it('shows tooltips for translation, pronunciation, and appearance controls', async () => {
    renderOverlay()

    fireEvent.mouseOver(screen.getByTestId('lyrics-toggle-translation'))
    expect(await screen.findByText('Toggle translations')).toBeInTheDocument()

    fireEvent.mouseOver(screen.getByTestId('lyrics-toggle-pronunciation'))
    expect(await screen.findByText('Toggle pronunciations')).toBeInTheDocument()

    fireEvent.mouseOver(screen.getByTestId('lyrics-settings-button'))
    expect(await screen.findByText('Appearance')).toBeInTheDocument()
  })

  it('renders the appearance popup with Main label and default line height for older settings', async () => {
    localStorage.setItem(
      'karaoke-lyrics-settings',
      JSON.stringify({
        tr: { fontSize: 16, colorKey: 'blue' },
        main: { fontSize: 26, colorKey: 'white' },
        pr: { fontSize: 15, colorKey: 'green' },
      }),
    )

    renderOverlay()

    fireEvent.click(screen.getByTestId('lyrics-settings-button'))

    expect(await screen.findByText('Appearance')).toBeInTheDocument()
    expect(screen.getByText('Main', { selector: 'div' })).toBeInTheDocument()
    expect(screen.queryByText('Default')).not.toBeInTheDocument()
    expect(screen.getByTestId('lyrics-reset-appearance')).toBeInTheDocument()
    expect(screen.getByTestId('lyrics-line-height-value')).toHaveTextContent(
      DEFAULT_LINE_HEIGHT_TEXT,
    )
  })

  it('renders the lyric group in main, pronunciation, translation order with layer badges', () => {
    renderOverlay({
      showTranslation: true,
      showPronunciation: true,
    })

    const mainLine = screen.getByText('こんにちは')
    const pronunciationLine = screen.getByText('konnichiwa')
    const translationLine = screen.getByText('Hello')

    expect(
      mainLine.compareDocumentPosition(pronunciationLine) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      pronunciationLine.compareDocumentPosition(translationLine) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()

    expect(screen.getByTestId('lyrics-language-badge-main')).toHaveTextContent(
      'Mainja',
    )
    expect(screen.getByTestId('lyrics-language-badge-pr')).toHaveTextContent(
      'PRja-Latn',
    )
    expect(screen.getByTestId('lyrics-language-badge-tr')).toHaveTextContent(
      'TRen',
    )
  })

  it('renders line-timed rows as whole-line spans without synthetic token splits', () => {
    renderOverlay({
      mainLyric: {
        kind: 'main',
        lang: 'en',
        synced: true,
        line: [
          { start: 1000, end: 2400, value: 'Batter up, batter up, batter up' },
        ],
      },
      translationLyric: {
        kind: 'translation',
        lang: 'ja',
        synced: true,
        line: [
          {
            start: 1000,
            end: 2400,
            value: 'バッターアップ、バッターアップ、バッターアップ',
          },
        ],
      },
      pronunciationLyric: {
        kind: 'pronunciation',
        lang: 'ja-Latn',
        synced: true,
        line: [
          {
            start: 1000,
            end: 2400,
            value: 'Battaa appu, battaa appu, battaa appu',
          },
        ],
      },
      showTranslation: true,
      showPronunciation: true,
    })

    const mainLine = screen.getByText(
      'Batter up, batter up, batter up',
    ).parentElement
    const pronunciationLine = screen.getByText(
      'Battaa appu, battaa appu, battaa appu',
    ).parentElement
    const translationLine = screen.getByText(
      'バッターアップ、バッターアップ、バッターアップ',
    ).parentElement

    expect(mainLine.querySelectorAll('span')).toHaveLength(1)
    expect(pronunciationLine.querySelectorAll('span')).toHaveLength(1)
    expect(translationLine.querySelectorAll('span')).toHaveLength(1)
  })

  it('highlights line-timed pronunciation and translation rows with the active main line', () => {
    renderOverlay({
      mainLyric: {
        kind: 'main',
        lang: 'en',
        synced: true,
        line: [
          { start: 1000, end: 1800, value: 'Line one' },
          { start: 2500, end: 3300, value: 'Line two' },
        ],
      },
      translationLyric: {
        kind: 'translation',
        lang: 'ja',
        synced: true,
        line: [
          { start: 1000, end: 1800, value: '一行目' },
          { start: 2500, end: 3300, value: '二行目' },
        ],
      },
      pronunciationLyric: {
        kind: 'pronunciation',
        lang: 'ja-Latn',
        synced: true,
        line: [
          { start: 1000, end: 1800, value: 'ichigyoume' },
          { start: 2500, end: 3300, value: 'nigyoume' },
        ],
      },
      showTranslation: true,
      showPronunciation: true,
      audioInstance: {
        ...audioInstance,
        currentTime: 1.2,
      },
    })

    const activePronunciation = screen.getByText('ichigyoume').parentElement
    const inactivePronunciation = screen.getByText('nigyoume').parentElement
    const activeTranslation = screen.getByText('一行目').parentElement
    const inactiveTranslation = screen.getByText('二行目').parentElement

    expect(parseFloat(activePronunciation.style.opacity)).toBeGreaterThan(
      parseFloat(inactivePronunciation.style.opacity),
    )
    expect(parseFloat(activeTranslation.style.opacity)).toBeGreaterThan(
      parseFloat(inactiveTranslation.style.opacity),
    )
  })

  it('renders untimed text lyrics in manual reading mode without a pinned active line', () => {
    renderOverlay({
      mainLyric: {
        kind: 'main',
        lang: 'en',
        synced: false,
        line: [{ value: 'First plain line' }, { value: 'Second plain line' }],
      },
      translationLyric: null,
      pronunciationLyric: null,
      showTranslation: false,
      showPronunciation: false,
      translationEnabled: false,
      pronunciationEnabled: false,
    })

    const firstLine = screen.getByText('First plain line').parentElement
    const secondLine = screen.getByText('Second plain line').parentElement

    expect(firstLine.style.opacity).toBe('1')
    expect(secondLine.style.opacity).toBe('1')
    expect(firstLine.style.color).toBe(secondLine.style.color)
  })

  it('persists line height changes, keeps aux line spacing fixed, and stores overlay height', async () => {
    renderOverlay({
      mainLyric: buildLyric('main', 'en', 'Hello world'),
      translationLyric: buildLyric('translation', 'es', 'Hola'),
      pronunciationLyric: buildLyric('pronunciation', 'en-Latn', 'heh-loh'),
      showTranslation: true,
      showPronunciation: true,
      translationEnabled: true,
      pronunciationEnabled: true,
    })

    const overlay = screen.getByTestId('karaoke-lyrics-overlay')
    const mainLine = screen.getByText('Hello world').parentElement
    const pronunciationLine = screen.getByText('heh-loh').parentElement
    expect(mainLine).toHaveStyle(`line-height: ${DEFAULT_LINE_HEIGHT_TEXT}`)
    expect(pronunciationLine).toHaveStyle('line-height: 1.2')

    fireEvent.click(screen.getByTestId('lyrics-settings-button'))

    const slider = screen.getByRole('slider', { name: 'Line height' })
    slider.focus()
    fireEvent.keyDown(slider, { key: 'ArrowRight' })

    await waitFor(() =>
      expect(screen.getByTestId('lyrics-line-height-value')).toHaveTextContent(
        NEXT_LINE_HEIGHT_TEXT,
      ),
    )

    await waitFor(() =>
      expect(mainLine).toHaveStyle(`line-height: ${NEXT_LINE_HEIGHT_TEXT}`),
    )
    expect(pronunciationLine).toHaveStyle('line-height: 1.2')

    fireEvent.mouseDown(screen.getByTestId('lyrics-resize-handle'), {
      clientY: 400,
    })
    fireEvent.mouseMove(window, { clientY: 360 })
    fireEvent.mouseUp(window)

    await waitFor(() => expect(overlay).toHaveStyle('height: 340px'))

    const stored = JSON.parse(localStorage.getItem('karaoke-lyrics-settings'))
    expect(stored.lineHeight).toBeCloseTo(1.32, 2)
    expect(stored.overlayHeight).toBe(340)
  })

  it('resets appearance back to the default spacing and overlay height', async () => {
    localStorage.setItem(
      'karaoke-lyrics-settings',
      JSON.stringify({
        lineHeight: 1.8,
        overlayHeight: 420,
        tr: { fontSize: 16, colorKey: 'yellow' },
        main: { fontSize: 28, colorKey: 'cyan' },
        pr: { fontSize: 15, colorKey: 'pink' },
      }),
    )

    renderOverlay({
      mainLyric: buildLyric('main', 'en', 'Hello world'),
      translationLyric: null,
      pronunciationLyric: null,
      showPronunciation: false,
      translationEnabled: false,
      pronunciationEnabled: false,
    })

    const overlay = screen.getByTestId('karaoke-lyrics-overlay')
    const mainLine = screen.getByText('Hello world').parentElement
    expect(overlay).toHaveStyle('height: 420px')
    expect(mainLine).toHaveStyle('line-height: 1.8')

    fireEvent.click(screen.getByTestId('lyrics-settings-button'))
    fireEvent.click(screen.getByTestId('lyrics-reset-appearance'))

    await waitFor(() =>
      expect(screen.getByTestId('lyrics-line-height-value')).toHaveTextContent(
        DEFAULT_LINE_HEIGHT_TEXT,
      ),
    )
    await waitFor(() => expect(overlay).toHaveStyle('height: 300px'))
    await waitFor(() =>
      expect(mainLine).toHaveStyle(`line-height: ${DEFAULT_LINE_HEIGHT_TEXT}`),
    )

    const stored = JSON.parse(localStorage.getItem('karaoke-lyrics-settings'))
    expect(stored.lineHeight).toBeCloseTo(1.3, 2)
    expect(stored.overlayHeight).toBe(300)
  })
})
