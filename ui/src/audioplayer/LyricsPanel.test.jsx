import React from 'react'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LyricsPanel from './LyricsPanel'
import {
  KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO,
  KARAOKE_MANUAL_SCROLL_PAUSE_MS,
} from './lyricsKaraokeConstants'
import { buildSegmentsFromLine } from './lyricsSegments'

const theme = createTheme({
  palette: {
    primary: { main: '#2266aa' },
    text: { primary: '#111111', secondary: '#778899' },
  },
})

const renderPanel = (props) =>
  render(
    <ThemeProvider theme={theme}>
      <LyricsPanel visible {...props} />
    </ThemeProvider>,
  )

const getRgbaAlpha = (value) => Number(value.match(/,\s*([\d.]+)\)$/)?.[1])

const mainLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'Main line' }],
}

const tokenizedMainLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'Main line' }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value: 'Main line',
      cue: [
        { start: 0, end: 500, value: 'Main', byteStart: 0, byteEnd: 3 },
        { start: 500, end: 1000, value: 'line', byteStart: 5, byteEnd: 8 },
      ],
    },
  ],
}

const tokenizedPronunciationLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'mein lain' }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value: 'mein lain',
      cue: [
        { start: 0, end: 500, value: 'mein', byteStart: 0, byteEnd: 3 },
        { start: 500, end: 1000, value: 'lain', byteStart: 5, byteEnd: 8 },
      ],
    },
  ],
}

const lineLevelChineseLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: '我总要给一些别的' }],
}

const lineLevelPinyinLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'wo zong yao gei yi xie bie de' }],
}

const multilineLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'first line\nsecond line' }],
}

const tokenizedKoreanMainLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: '너와 함께' }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value: '너와 함께',
      cue: [
        { start: 0, end: 500, value: '너와', byteStart: 0, byteEnd: 5 },
        { start: 500, end: 1000, value: '함께', byteStart: 7, byteEnd: 12 },
      ],
    },
  ],
}

const tokenizedKoreanPronunciationLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: 'neo wa ham kke' }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value: 'neo wa ham kke',
      cue: [
        { start: 0, end: 500, value: 'neo wa', byteStart: 0, byteEnd: 5 },
        {
          start: 500,
          end: 1000,
          value: 'ham kke',
          byteStart: 7,
          byteEnd: 13,
        },
      ],
    },
  ],
}

const emphasisRoleLyric = {
  synced: true,
  line: [{ start: 0, end: 1000, value: '(Us,hello?)' }],
  cueLine: [
    {
      index: 0,
      start: 0,
      end: 1000,
      value: '(Us,hello?)',
      role: 'chorus',
      cue: [
        {
          start: 0,
          end: 1000,
          value: '(Us,hello?)',
          byteStart: 0,
          byteEnd: 10,
        },
      ],
    },
  ],
}

const multiAgentLyric = {
  synced: true,
  agents: [
    { id: 'lead', role: 'main' },
    { id: 'all', role: 'group' },
    { id: 'echo', role: 'bg' },
  ],
  line: [{ start: 1000, end: 4000, value: 'Lead all echo' }],
  cueLine: [
    {
      index: 0,
      start: 1000,
      end: 2000,
      value: 'Lead',
      agentId: 'lead',
      cue: [{ start: 1000, end: 2000, value: 'Lead' }],
    },
    {
      index: 0,
      start: 1500,
      end: 2600,
      value: 'all',
      agentId: 'all',
      cue: [{ start: 1500, end: 2600, value: 'all' }],
    },
    {
      index: 0,
      start: 2200,
      end: 3400,
      value: 'echo',
      agentId: 'echo',
      cue: [{ start: 2200, end: 3400, value: 'echo' }],
    },
  ],
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

    expect(screen.getAllByTestId('lyrics-pronunciation-token')).toHaveLength(2)
    expect(
      screen.getAllByTestId('lyrics-pronunciation-token')[0],
    ).toHaveTextContent('mein')
    expect(
      screen.getAllByTestId('lyrics-pronunciation-token')[1],
    ).toHaveTextContent('lain')
    expect(screen.getByText('translation line')).toBeInTheDocument()
  })

  it('keeps full line-level pronunciation when main lyric has no word boundaries', () => {
    renderPanel({
      mainLyric: lineLevelChineseLyric,
      pronunciationLyric: lineLevelPinyinLyric,
      showPronunciation: true,
    })

    expect(screen.getByText('我总要给一些别的')).toBeInTheDocument()

    const pronunciationTokens = screen.getAllByTestId(
      'lyrics-pronunciation-token',
    )
    expect(pronunciationTokens).toHaveLength(1)
    expect(pronunciationTokens[0]).toHaveTextContent(
      'wo zong yao gei yi xie bie de',
    )
  })

  it('allows long stacked pronunciation text to wrap inside the sidebar', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: '到手的数字甚至懒得计算了' }],
      },
      pronunciationLyric: {
        synced: true,
        line: [
          {
            start: 0,
            end: 1000,
            value: 'dao shou de shu zi shen zhi lan de ji suan le',
          },
        ],
      },
      showPronunciation: true,
    })

    const pronunciation = screen.getByText(
      'dao shou de shu zi shen zhi lan de ji suan le',
    )
    const style = window.getComputedStyle(pronunciation)
    expect(style.whiteSpace).toBe('pre-wrap')
    expect(style.overflowWrap).toBe('anywhere')
  })

  it('renders unsynced plain text without active or scroll highlight', () => {
    renderPanel({
      mainLyric: {
        synced: false,
        line: [{ value: 'first plain line' }, { value: 'second plain line' }],
      },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    expect(groups).toHaveLength(2)
    expect(groups[0]).toHaveAttribute('data-active', 'false')
    expect(groups[0]).not.toHaveAttribute('aria-current')
    expect(groups[0]).toHaveAttribute('data-scroll-target', 'false')
    expect(groups[1]).toHaveAttribute('data-active', 'false')
    expect(groups[1]).not.toHaveAttribute('aria-current')
    expect(groups[1]).toHaveAttribute('data-scroll-target', 'false')
  })

  it('preserves explicit line breaks in rendered lyric text', () => {
    renderPanel({ mainLyric: multilineLyric })

    const line = screen
      .getByTestId('lyrics-line-group')
      .querySelector('.MuiTypography-root')
    expect(line).toBeInTheDocument()
    expect(line.textContent).toBe('first line\nsecond line')
    expect(window.getComputedStyle(line).whiteSpace).toBe('pre-wrap')
  })

  it('derives line colors from the current Material UI theme', () => {
    renderPanel({
      mainLyric,
      pronunciationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'pronunciation line' }],
      },
      translationLyric: {
        synced: true,
        line: [{ start: 0, end: 1000, value: 'translation line' }],
      },
      showPronunciation: false,
      showTranslation: true,
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    expect(screen.getByText('Main line')).toHaveStyle({
      color: 'rgba(17, 17, 17, 0.98)',
    })
    expect(screen.getByText('translation line')).toHaveStyle({
      color: 'rgba(119, 136, 153, 0.72)',
    })
  })

  it('eases translation emphasis without moving the translation row independently', () => {
    const lyric = {
      synced: true,
      line: [
        { start: 0, end: 1000, value: 'first line' },
        { start: 1000, end: 2000, value: 'second line' },
      ],
    }
    const translationLyric = {
      synced: true,
      line: [
        { start: 0, end: 1000, value: 'first translation' },
        { start: 1000, end: 2000, value: 'second translation' },
      ],
    }

    const { rerender } = renderPanel({
      mainLyric: lyric,
      translationLyric,
      showTranslation: true,
      audioInstance: {
        currentTime: 1.02,
        paused: true,
      },
    })

    const earlyTranslation = screen
      .getByText('second translation')
      .closest('.MuiTypography-root')
    const earlyGroup = earlyTranslation.closest(
      '[data-testid="lyrics-line-group"]',
    )
    const earlyAlpha = getRgbaAlpha(earlyTranslation.style.color)
    expect(earlyAlpha).toBeGreaterThan(0.34)
    expect(earlyAlpha).toBeLessThan(0.72)
    expect(earlyTranslation.style.transform).toBe('')
    expect(earlyGroup.style.transform).not.toBe(
      'scale(1.000) translateY(0.00px)',
    )

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={lyric}
          translationLyric={translationLyric}
          showTranslation
          audioInstance={{
            currentTime: 1.25,
            paused: true,
          }}
        />
      </ThemeProvider>,
    )

    const settledTranslation = screen
      .getByText('second translation')
      .closest('.MuiTypography-root')
    const settledGroup = settledTranslation.closest(
      '[data-testid="lyrics-line-group"]',
    )
    expect(getRgbaAlpha(settledTranslation.style.color)).toBe(0.72)
    expect(settledTranslation.style.transform).toBe('')
    expect(settledGroup.style.transform).toBe('')
  })

  it('highlights stacked pronunciation tokens from the current Material UI theme', () => {
    renderPanel({
      mainLyric: tokenizedMainLyric,
      pronunciationLyric: tokenizedPronunciationLyric,
      showPronunciation: true,
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    expect(screen.getByText('mein').style.color).toBe('transparent')
    expect(screen.getByText('mein').style.backgroundImage).toContain(
      'linear-gradient',
    )
    expect(screen.getByText('mein').style.textShadow).toBe('')
  })

  it('does not pre-highlight future stacked pronunciation tokens', () => {
    renderPanel({
      mainLyric: tokenizedMainLyric,
      pronunciationLyric: tokenizedPronunciationLyric,
      showPronunciation: true,
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    expect(screen.getByText('lain').style.backgroundImage).toBe('none')
    expect(screen.getByText('lain')).toHaveStyle({
      color: 'rgba(34, 102, 170, 0.34)',
    })
  })

  it('nudges token highlighting slightly ahead without activating the next line early', () => {
    renderPanel({
      mainLyric: tokenizedMainLyric,
      audioInstance: {
        currentTime: 0.43,
        paused: true,
      },
    })

    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-active',
      'true',
    )
    expect(screen.getByText('line').style.backgroundImage).toContain(
      'linear-gradient',
    )
  })

  it('renders timed pronunciation spans for tokenized pronunciation lyrics', () => {
    renderPanel({
      mainLyric: tokenizedKoreanMainLyric,
      pronunciationLyric: tokenizedKoreanPronunciationLyric,
      showPronunciation: true,
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    const pronunciationTokens = screen.getAllByTestId(
      'lyrics-pronunciation-token',
    )
    expect(pronunciationTokens).toHaveLength(2)
    expect(pronunciationTokens[0]).toHaveTextContent('neo wa')
    expect(pronunciationTokens[0].style.backgroundImage).toContain(
      'linear-gradient',
    )
    expect(pronunciationTokens[1]).toHaveTextContent('ham kke')
    expect(pronunciationTokens[1].style.backgroundImage).toBe('none')
  })

  it('adds stacked pronunciation row spacing only after wrapping', async () => {
    const originalOffsetTop = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'offsetTop',
    )

    Object.defineProperty(HTMLElement.prototype, 'offsetTop', {
      configurable: true,
      get() {
        return this.textContent?.includes('lain') ? 20 : 0
      },
    })

    try {
      renderPanel({
        mainLyric: tokenizedMainLyric,
        pronunciationLyric: tokenizedPronunciationLyric,
        showPronunciation: true,
      })

      await waitFor(() => {
        expect(
          screen.getByText('mein').closest('[data-wrapped]'),
        ).toHaveAttribute('data-wrapped', 'true')
      })
    } finally {
      if (originalOffsetTop) {
        Object.defineProperty(
          HTMLElement.prototype,
          'offsetTop',
          originalOffsetTop,
        )
      } else {
        delete HTMLElement.prototype.offsetTop
      }
    }
  })

  it('keeps active and inactive line metrics stable during highlighting', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 0, end: 1000, value: 'Active line' },
          { start: 1000, end: 2000, value: 'Next line' },
        ],
      },
    })

    expect(screen.getByText('Active line').style.fontSize).toBe('')
    expect(screen.getByText('Active line').style.maxWidth).toBe('')
    expect(screen.getByText('Next line').style.fontSize).toBe('')
    expect(screen.getByText('Next line').style.maxWidth).toBe('')
  })

  it('adds bottom scroll room so the last active line can keep the anchor position', () => {
    const originalClientHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'clientHeight',
    )

    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get() {
        return this.getAttribute('data-testid') === 'lyrics-scroll-body'
          ? 500
          : 0
      },
    })

    try {
      renderPanel({
        mainLyric: {
          synced: true,
          line: [
            { start: 0, end: 1000, value: 'First line' },
            { start: 1000, end: 2000, value: 'Last line' },
          ],
        },
        audioInstance: {
          currentTime: 1.2,
          paused: true,
        },
      })

      const lines = screen
        .getByTestId('lyrics-scroll-body')
        .querySelector('[data-scroll-end-padding]')
      const expectedPadding = Math.round(
        500 * (1 - KARAOKE_DESKTOP_ACTIVE_LINE_ANCHOR_RATIO),
      )
      expect(lines).toHaveAttribute(
        'data-scroll-end-padding',
        String(expectedPadding),
      )
      expect(lines).toHaveStyle({ paddingBottom: `${expectedPadding}px` })
    } finally {
      if (originalClientHeight) {
        Object.defineProperty(
          HTMLElement.prototype,
          'clientHeight',
          originalClientHeight,
        )
      } else {
        delete HTMLElement.prototype.clientHeight
      }
    }
  })

  it('uses a lower inline anchor for mobile lyrics', () => {
    const originalClientHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      'clientHeight',
    )

    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
      configurable: true,
      get() {
        return this.getAttribute('data-testid') === 'lyrics-scroll-body'
          ? 500
          : 0
      },
    })

    try {
      renderPanel({
        inline: true,
        mainLyric: {
          synced: true,
          line: [
            { start: 0, end: 1000, value: 'First line' },
            { start: 1000, end: 2000, value: 'Last line' },
          ],
        },
        audioInstance: {
          currentTime: 1.2,
          paused: true,
        },
      })

      const lines = screen
        .getByTestId('lyrics-scroll-body')
        .querySelector('[data-scroll-end-padding]')
      expect(lines).toHaveAttribute('data-scroll-end-padding', '290')
      expect(lines).toHaveStyle({ paddingBottom: '290px' })
    } finally {
      if (originalClientHeight) {
        Object.defineProperty(
          HTMLElement.prototype,
          'clientHeight',
          originalClientHeight,
        )
      } else {
        delete HTMLElement.prototype.clientHeight
      }
    }
  })

  it('exposes the active line without changing line metrics', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 0, end: 1000, value: 'Current line' },
          { start: 1000, end: 2000, value: 'Later line' },
        ],
      },
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    const activeGroup = screen
      .getByText('Current line')
      .closest('[data-testid="lyrics-line-group"]')
    expect(activeGroup).toHaveAttribute('data-active', 'true')
    expect(activeGroup).toHaveAttribute('aria-current', 'true')
    expect(activeGroup.style.transform).toBe('')
    expect(screen.getByText('Current line').style.fontSize).toBe('')
    expect(screen.getByText('Current line').style.transform).toBe('')
    expect(screen.getByText('Later line').style.fontSize).toBe('')
    expect(screen.getByText('Current line').style.textShadow).toBe('')
    expect(screen.getByText('Later line').style.textShadow).toBe('')
  })

  it('honors reduced motion preference for scroll behavior', async () => {
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }))

    renderPanel({ mainLyric })

    await waitFor(() => {
      expect(screen.getByTestId('lyrics-scroll-body')).toHaveAttribute(
        'data-reduced-motion',
        'true',
      )
    })
  })

  it('enables the top fade only after the lyrics body scrolls down', () => {
    renderPanel({ mainLyric })

    const body = screen.getByTestId('lyrics-scroll-body')
    expect(body).toHaveAttribute('data-top-fade-enabled', 'false')

    body.scrollTop = 8
    fireEvent.scroll(body)
    expect(body).toHaveAttribute('data-top-fade-enabled', 'true')

    body.scrollTop = 0
    fireEvent.scroll(body)
    expect(body).toHaveAttribute('data-top-fade-enabled', 'false')
  })

  it('resets scroll to the top when the lyric content changes', () => {
    const { rerender } = renderPanel({ mainLyric })
    const body = screen.getByTestId('lyrics-scroll-body')
    body.scrollTop = 180

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={{
            synced: true,
            line: [{ start: 0, end: 1000, value: 'Different song' }],
          }}
        />
      </ThemeProvider>,
    )

    expect(body.scrollTop).toBe(0)
  })

  it('shows the scrollbar temporarily after manual scroll intent', () => {
    vi.useFakeTimers()
    renderPanel({ mainLyric })

    const body = screen.getByTestId('lyrics-scroll-body')
    expect(body).toHaveAttribute('data-scrollbar-visible', 'false')

    fireEvent.wheel(body)

    expect(body).toHaveAttribute('data-scrollbar-visible', 'true')

    vi.advanceTimersByTime(1400)

    expect(body).toHaveAttribute('data-scrollbar-visible', 'false')
  })

  it('resumes auto-scroll after manual pause even when active line is unchanged', async () => {
    vi.useFakeTimers()
    const requestAnimationFrameSpy = vi
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation(() => 0)
    renderPanel({
      mainLyric,
      audioInstance: {
        currentTime: 0.5,
        paused: true,
      },
    })

    const body = screen.getByTestId('lyrics-scroll-body')
    const initialFrameCount = requestAnimationFrameSpy.mock.calls.length

    fireEvent.wheel(body)

    act(() => {
      vi.advanceTimersByTime(KARAOKE_MANUAL_SCROLL_PAUSE_MS)
    })

    await waitFor(() => {
      expect(requestAnimationFrameSpy.mock.calls.length).toBeGreaterThan(
        initialFrameCount,
      )
    })
  })

  it('holds the finished line during long pauses until the next line pre-roll', async () => {
    const lyric = {
      synced: true,
      line: [
        { start: 0, end: 1000, value: 'Finished line' },
        { start: 2000, end: 3000, value: 'Upcoming line' },
      ],
    }

    const { rerender } = renderPanel({
      mainLyric: lyric,
      audioInstance: {
        currentTime: 1.2,
        paused: true,
      },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    await waitFor(() => {
      expect(groups[0]).toHaveAttribute('data-scroll-target', 'true')
    })
    expect(groups[0]).toHaveAttribute('data-active', 'false')
    expect(groups[1]).toHaveAttribute('data-active', 'false')
    const upcomingLineBefore = screen
      .getByText('Upcoming line')
      .closest('.MuiTypography-root')
    const upcomingStyleBefore = {
      opacity: upcomingLineBefore.style.opacity,
      color: upcomingLineBefore.style.color,
      groupTransform: groups[1].style.transform,
    }

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={lyric}
          audioInstance={{
            currentTime: 1.7,
            paused: true,
          }}
        />
      </ThemeProvider>,
    )

    expect(groups[1]).toHaveAttribute('data-scroll-target', 'true')
    expect(groups[1]).toHaveAttribute('data-active', 'false')
    expect(groups[1]).toHaveAttribute('data-lifecycle', 'idle')
    expect(groups[1]).toHaveAttribute('data-highlight-active', 'false')
    expect(
      screen.getByText('Upcoming line').closest('.MuiTypography-root'),
    ).toHaveStyle({
      opacity: '1',
      color: upcomingStyleBefore.color,
    })
    expect(upcomingStyleBefore.opacity).toBe('1')
    expect(groups[1].style.transform).toBe(upcomingStyleBefore.groupTransform)
  })

  it('skips untimed lines while choosing the finished scroll target', async () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 0, end: 1000, value: 'Opening line' },
          { value: '[instrumental]' },
          { start: 5000, end: 6000, value: 'Later line' },
        ],
      },
      audioInstance: {
        currentTime: 6.5,
        paused: true,
      },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')

    await waitFor(() => {
      expect(groups[2]).toHaveAttribute('data-scroll-target', 'true')
    })
    expect(groups[0]).toHaveAttribute('data-scroll-target', 'false')
  })

  it('crossfades very short tokens instead of drawing a hard wipe', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 0, end: 120, value: 'go' }],
        cueLine: [
          {
            index: 0,
            start: 0,
            end: 120,
            value: 'go',
            cue: [{ start: 0, end: 120, value: 'go' }],
          },
        ],
      },
      audioInstance: {
        currentTime: 0.02,
        paused: true,
      },
    })

    const token = screen.getByTestId('lyrics-token')
    expect(token).toHaveTextContent('go')
    expect(token.style.backgroundImage).toBe('none')
    expect(token.style.color).toMatch(/^rgba\(17, 17, 17,/)
  })

  it('fades completed highlighting during release after the line becomes inactive', () => {
    const { rerender } = renderPanel({
      mainLyric: tokenizedMainLyric,
      pronunciationLyric: tokenizedPronunciationLyric,
      showPronunciation: true,
      audioInstance: {
        currentTime: 0.25,
        paused: true,
      },
    })

    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-active',
      'true',
    )
    expect(screen.getByText('Main').style.color).toBe('transparent')
    expect(screen.getByText('Main').style.backgroundImage).toContain(
      'linear-gradient',
    )
    expect(screen.getByText('mein').style.color).toBe('transparent')
    expect(screen.getByText('mein').style.backgroundImage).toContain(
      'linear-gradient',
    )

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={tokenizedMainLyric}
          pronunciationLyric={tokenizedPronunciationLyric}
          showPronunciation
          audioInstance={{
            currentTime: 1.1,
            paused: true,
          }}
        />
      </ThemeProvider>,
    )

    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-active',
      'false',
    )
    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-lifecycle',
      'release',
    )
    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-highlight-active',
      'true',
    )
    expect(screen.getAllByTestId('lyrics-line-group')[0]).not.toHaveAttribute(
      'data-exiting',
    )
    expect(getRgbaAlpha(screen.getByText('Main').style.color)).toBeGreaterThan(
      0.34,
    )
    expect(getRgbaAlpha(screen.getByText('Main').style.color)).toBeLessThan(1)
    expect(screen.getByText('Main').style.backgroundImage).toBe('none')
    expect(screen.getByText('Main').style.textShadow).toBe('')
    expect(getRgbaAlpha(screen.getByText('mein').style.color)).toBeGreaterThan(
      0.34,
    )
    expect(getRgbaAlpha(screen.getByText('mein').style.color)).toBeLessThan(1)
    expect(screen.getByText('mein').style.backgroundImage).toBe('none')
    expect(screen.getByText('mein').style.textShadow).toBe('')

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={tokenizedMainLyric}
          pronunciationLyric={tokenizedPronunciationLyric}
          showPronunciation
          audioInstance={{
            currentTime: 1.25,
            paused: true,
          }}
        />
      </ThemeProvider>,
    )

    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-lifecycle',
      'idle',
    )
    expect(screen.getAllByTestId('lyrics-line-group')[0]).toHaveAttribute(
      'data-highlight-active',
      'false',
    )
    expect(screen.getByText('Main').style.color).not.toBe('transparent')
    expect(screen.getByText('Main').style.backgroundImage).toBe('none')
    expect(screen.getByText('mein').style.color).not.toBe('transparent')
    expect(screen.getByText('mein').style.backgroundImage).toBe('none')
  })

  it('keeps background and chorus-style tokens italic after highlighting ends', async () => {
    const { rerender } = renderPanel({
      mainLyric: emphasisRoleLyric,
      audioInstance: {
        currentTime: 0.2,
        paused: true,
      },
    })

    expect(screen.getByText('(Us,hello?)').style.fontStyle).toBe('italic')
    expect(screen.getByText('(Us,hello?)').style.backgroundImage).toContain(
      'linear-gradient',
    )

    rerender(
      <ThemeProvider theme={theme}>
        <LyricsPanel
          visible
          mainLyric={emphasisRoleLyric}
          audioInstance={{
            currentTime: 1.6,
            paused: true,
          }}
        />
      </ThemeProvider>,
    )

    expect(screen.getAllByTestId('lyrics-line-group')[0]).not.toHaveAttribute(
      'data-exiting',
    )
    expect(screen.getByText('(Us,hello?)').style.fontStyle).toBe('italic')
    expect(screen.getByText('(Us,hello?)').style.backgroundImage).toBe('none')
    expect(screen.getByText('(Us,hello?)').style.textShadow).toBe('')
  })

  it('keeps same-index agent cue lines grouped as separate italic voice lanes', () => {
    renderPanel({
      mainLyric: multiAgentLyric,
      audioInstance: {
        currentTime: 4.5,
        paused: true,
      },
    })

    expect(screen.getByTestId('lyrics-voice-lanes')).toBeInTheDocument()
    const lanes = screen.getAllByTestId('lyrics-voice-lane')
    expect(lanes).toHaveLength(3)
    expect(lanes[0]).toHaveTextContent('Lead')
    expect(lanes[0].style.fontStyle).toBe('')
    expect(lanes[1]).toHaveTextContent('all')
    expect(lanes[1].style.fontStyle).toBe('italic')
    expect(lanes[2]).toHaveTextContent('echo')
    expect(lanes[2].style.fontStyle).toBe('italic')
    expect(screen.getByTestId('lyrics-line-group')).toHaveAttribute(
      'data-active',
      'false',
    )
  })

  it('highlights overlapping active lines while keeping one primary scroll target', () => {
    renderPanel({
      mainLyric: {
        synced: true,
        line: [
          { start: 1000, end: 4000, value: 'Lead vocal' },
          { start: 2000, end: 3000, value: 'Answer vocal' },
          { start: 5000, end: 6000, value: 'Later vocal' },
        ],
      },
      audioInstance: {
        currentTime: 2.5,
        paused: true,
      },
    })

    const groups = screen.getAllByTestId('lyrics-line-group')
    expect(groups[0]).toHaveAttribute('data-active', 'true')
    expect(groups[1]).toHaveAttribute('data-active', 'true')
    expect(groups[2]).toHaveAttribute('data-active', 'false')
    expect(groups[0]).not.toHaveAttribute('aria-current')
    expect(groups[1]).toHaveAttribute('aria-current', 'true')
    expect(groups[1]).toHaveAttribute('data-scroll-target', 'true')
  })

  it('seeks to the clicked line start time', () => {
    const audioInstance = { currentTime: 0 }
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 2300, end: 3200, value: 'Seek line' }],
      },
      showPronunciation: true,
      showTranslation: true,
      audioInstance,
    })

    const line = screen.getByText('Seek line')
    const group = line.closest('[data-testid="lyrics-line-group"]')
    expect(group).toHaveAttribute('role', 'button')
    expect(group).toHaveAttribute('tabindex', '0')

    fireEvent.click(line)

    expect(audioInstance.currentTime).toBe(2.3)

    audioInstance.currentTime = 0
    fireEvent.keyDown(group, { key: 'Enter' })
    expect(audioInstance.currentTime).toBe(2.3)

    audioInstance.currentTime = 0
    fireEvent.keyDown(group, { key: ' ' })
    expect(audioInstance.currentTime).toBe(2.3)
  })

  it('does not keep focus on a mouse-clicked lyric line', async () => {
    const user = userEvent.setup()
    const audioInstance = { currentTime: 0 }
    renderPanel({
      mainLyric: {
        synced: true,
        line: [{ start: 2300, end: 3200, value: 'Mouse seek line' }],
      },
      audioInstance,
    })

    const group = screen
      .getByText('Mouse seek line')
      .closest('[data-testid="lyrics-line-group"]')

    await user.click(group)

    expect(audioInstance.currentTime).toBe(2.3)
    expect(document.activeElement).not.toBe(group)
  })

  it('uses exact byte-offset segments for repeated UTF-8 tokens', () => {
    const text = 'caf\u00e9 caf\u00e9'
    const segments = buildSegmentsFromLine({
      value: text,
      tokens: [
        { value: 'caf\u00e9', byteStart: 0, byteEnd: 4 },
        { value: 'caf\u00e9', byteStart: 6, byteEnd: 10 },
      ],
    })

    expect(segments).toEqual([
      expect.objectContaining({ text: 'caf\u00e9', tokenIndex: 0 }),
      expect.objectContaining({ text: ' ', tokenIndex: -1 }),
      expect.objectContaining({ text: 'caf\u00e9', tokenIndex: 1 }),
    ])
  })

  it('does not render appearance customization controls', () => {
    renderPanel({ mainLyric })

    expect(
      screen.queryByTestId('lyrics-settings-button'),
    ).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/font size/i)).not.toBeInTheDocument()
  })

  it('does not add artificial spacer rows before or after lyrics', () => {
    const { container } = renderPanel({ mainLyric })

    expect(container.querySelector('[aria-hidden="true"]')).toBeNull()
  })
})
