import { shouldUseSimpleMobile } from './simpleMobile'

const DEFAULT_LABELS = {
  play: 'Play',
  pause: 'Pause',
  prev: 'Previous track',
  next: 'Next track',
  shuffle: 'Play random songs',
  full: 'Open full version',
  hint: 'Nothing playing',
  shuffleHint: 'Shuffle to start',
}

const queueIndex = (playerState) => {
  const queue = playerState?.queue || []
  const current = playerState?.current || {}
  if (current?.uuid) {
    const found = queue.findIndex((item) => item.uuid === current.uuid)
    if (found >= 0) {
      return found
    }
  }
  const saved = playerState?.savedPlayIndex
  if (Number.isInteger(saved) && saved >= 0) {
    return saved
  }
  return 0
}

export const playerSnapshot = (playerState, audio) => {
  const queue = playerState?.queue || []
  const current = playerState?.current || {}
  const idx = queueIndex(playerState)
  const queued = queue[idx] || queue[0] || {}
  const song = current.song || queued.song || {}
  const title = song.title || current.name || queued.name || ''
  const artist = song.artist || current.singer || queued.singer || ''
  const cover = current.cover || queued.cover || ''
  const hasTrack = queue.length > 0 || Boolean(title)
  const playing = !!(audio && audio.paused === false)
  const canPrev = hasTrack && idx > 0
  const canNext = hasTrack && idx < queue.length - 1
  return { title, artist, cover, hasTrack, playing, canPrev, canNext }
}

export const paintSimpleShell = (state, doc = document) => {
  if (typeof doc === 'undefined' || !doc.getElementById) {
    return
  }
  const shell = doc.getElementById('nd-static-simple')
  if (!shell) {
    return
  }

  const {
    title = '',
    artist = '',
    cover = '',
    hasTrack = false,
    playing = false,
    canPrev = false,
    canNext = false,
    labels = {},
  } = state || {}
  const text = { ...DEFAULT_LABELS, ...labels }

  shell.dataset.hasTrack = hasTrack ? '1' : '0'
  shell.dataset.playing = playing ? '1' : '0'

  const titleEl = doc.getElementById('nd-static-title')
  const artistEl = doc.getElementById('nd-static-artist')
  const coverEl = doc.getElementById('nd-static-cover')
  const playBtn = doc.getElementById('nd-static-play')
  const prevBtn = doc.getElementById('nd-static-prev')
  const nextBtn = doc.getElementById('nd-static-next')
  const shuffleBtn = doc.getElementById('nd-static-shuffle')
  const fullBtn = doc.getElementById('nd-static-full')
  const hintEl = doc.getElementById('nd-static-hint')
  const hintSubEl = doc.getElementById('nd-static-hint-sub')

  if (titleEl) {
    titleEl.textContent = title
  }
  if (artistEl) {
    artistEl.textContent = artist
  }

  if (coverEl) {
    if (cover) {
      coverEl.alt = title
      if (coverEl.getAttribute('src') !== cover) {
        coverEl.hidden = false
        coverEl.src = cover
      }
    } else {
      coverEl.removeAttribute('src')
      coverEl.alt = ''
      coverEl.hidden = true
    }
  }

  if (playBtn) {
    const playLabel = playing ? text.pause : text.play
    playBtn.setAttribute('aria-label', playLabel)
    playBtn.setAttribute('aria-pressed', playing ? 'true' : 'false')
    playBtn.disabled = !hasTrack
  }
  if (prevBtn) {
    prevBtn.setAttribute('aria-label', text.prev)
    prevBtn.disabled = !canPrev
  }
  if (nextBtn) {
    nextBtn.setAttribute('aria-label', text.next)
    nextBtn.disabled = !canNext
  }

  if (shuffleBtn) {
    shuffleBtn.textContent = text.shuffle
  }
  if (fullBtn) {
    fullBtn.textContent = text.full
  }
  if (hintEl) {
    hintEl.textContent = text.hint
  }
  if (hintSubEl) {
    hintSubEl.textContent = text.shuffleHint
  }
}

export const callPlayRandom = () => {
  if (
    typeof window !== 'undefined' &&
    typeof window.__ndPlayRandom === 'function'
  ) {
    window.__ndPlayRandom()
    return true
  }
  return false
}

const callPlayerMethod = (name) => {
  if (
    typeof window !== 'undefined' &&
    window.__ndPlayer &&
    typeof window.__ndPlayer[name] === 'function'
  ) {
    window.__ndPlayer[name]()
    return true
  }
  return false
}

export const callPlayerToggle = () => callPlayerMethod('toggle')
export const callPlayerPrev = () => callPlayerMethod('prev')
export const callPlayerNext = () => callPlayerMethod('next')

export const bindSimpleMobilePlayer = (audio) => {
  if (typeof window === 'undefined') {
    return
  }
  window.__ndPlayer = {
    getState: () => ({
      paused: !audio || audio.paused !== false,
    }),
    play: () => {
      if (!audio || typeof audio.play !== 'function') {
        return
      }
      return audio.play()
    },
    pause: () => {
      if (!audio || typeof audio.pause !== 'function') {
        return
      }
      audio.pause()
    },
    toggle: () => {
      if (!audio) {
        return
      }
      if (typeof audio.togglePlay === 'function') {
        audio.togglePlay()
        return
      }
      if (audio.paused) {
        audio.play()
      } else {
        audio.pause()
      }
    },
    prev: () => {
      if (audio && typeof audio.playPrev === 'function') {
        audio.playPrev()
      }
    },
    next: () => {
      if (audio && typeof audio.playNext === 'function') {
        audio.playNext()
      }
    },
  }
}

export const unbindSimpleMobilePlayer = () => {
  if (typeof window === 'undefined') {
    return
  }
  delete window.__ndPlayer
}

export const syncSimpleMobilePlayer = ({ playerState, audio, labels } = {}) => {
  if (!shouldUseSimpleMobile()) {
    return () => {}
  }
  bindSimpleMobilePlayer(audio)
  const paint = () =>
    paintSimpleShell({ ...playerSnapshot(playerState, audio), labels })
  paint()
  if (!audio || typeof audio.addEventListener !== 'function') {
    return () => {
      unbindSimpleMobilePlayer()
    }
  }
  audio.addEventListener('play', paint)
  audio.addEventListener('pause', paint)
  audio.addEventListener('ended', paint)
  return () => {
    audio.removeEventListener('play', paint)
    audio.removeEventListener('pause', paint)
    audio.removeEventListener('ended', paint)
    unbindSimpleMobilePlayer()
  }
}
