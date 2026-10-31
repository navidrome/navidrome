import subsonic from '../../subsonic'

export const CAST_SENDER_SDK_URL =
  'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1'

export const MAX_CAST_WINDOW_ITEMS = 40
export const MAX_CAST_SAMPLE_RATE = 96000
export const MAX_CAST_CHANNELS = 2

export const canLoadCastSdk = (
  appConfig = {},
  win = typeof window !== 'undefined' ? window : undefined,
) => {
  if (!win) {
    return false
  }
  if (win.cast?.framework?.CastContext && win.chrome?.cast?.media) {
    return true
  }
  if (appConfig?.enableExternalServices === false) {
    return false
  }
  const ua = win.navigator?.userAgent || ''
  return Boolean(win.chrome) || /Chrome\/\d+|Chromium\/\d+|CriOS\/\d+/i.test(ua)
}

export const DIRECT_PLAY_MIME_BY_SUFFIX = {
  flac: 'audio/flac',
  mp3: 'audio/mpeg',
  m4a: 'audio/mp4',
  mp4: 'audio/mp4',
  aac: 'audio/mp4',
  ogg: 'audio/ogg',
  oga: 'audio/ogg',
  opus: 'audio/ogg',
  wav: 'audio/wav',
  webm: 'audio/webm',
}

export const UNSUPPORTED_CAST_CODECS = new Set([
  'alac',
  'wma',
  'wmav1',
  'wmav2',
  'wmapro',
  'wmalossless',
  'ape',
  'wavpack',
  'wv',
  'dsd',
  'dsf',
  'dff',
  'dsd_lsbf',
  'dsd_msbf',
  'mpc',
  'musepack',
  'shorten',
  'shn',
  'tta',
  'aiff',
  'aif',
  'pcm_s24be',
])

export const toAbsoluteUrl = (urlOrPath, origin) => {
  if (!urlOrPath || typeof urlOrPath !== 'string') {
    return ''
  }
  const base =
    origin || (typeof window !== 'undefined' ? window.location.origin : '')
  if (!base) {
    return urlOrPath
  }
  try {
    return new URL(urlOrPath, base).toString()
  } catch {
    return urlOrPath
  }
}

export const canDirectPlayOnCast = (song = {}) => {
  const suffix = String(song.suffix || '')
    .trim()
    .toLowerCase()
  const codec = String(song.codec || '')
    .trim()
    .toLowerCase()

  if (!suffix || !DIRECT_PLAY_MIME_BY_SUFFIX[suffix]) {
    return false
  }
  if (codec && UNSUPPORTED_CAST_CODECS.has(codec)) {
    return false
  }
  if (song.sampleRate && Number(song.sampleRate) > MAX_CAST_SAMPLE_RATE) {
    return false
  }
  if (song.channels && Number(song.channels) > MAX_CAST_CHANNELS) {
    return false
  }
  return true
}

export const resolveCastStream = (item, origin) => {
  if (!item) {
    return { url: '', contentType: 'audio/mpeg', isLive: false }
  }

  if (item.isRadio) {
    const rawUrl = item.song?.streamUrl || item.musicSrc || ''
    return {
      url: toAbsoluteUrl(typeof rawUrl === 'string' ? rawUrl : '', origin),
      contentType: item.song?.contentType || 'audio/mpeg',
      isLive: true,
    }
  }

  const song = item.song || {}
  const trackId = item.trackId || song.mediaFileId || song.id
  if (!trackId) {
    return { url: '', contentType: 'audio/mpeg', isLive: false }
  }

  if (canDirectPlayOnCast(song)) {
    const suffix = String(song.suffix).trim().toLowerCase()
    const streamPath = subsonic.streamUrl(trackId, { format: 'raw' })
    return {
      url: toAbsoluteUrl(streamPath, origin),
      contentType: DIRECT_PLAY_MIME_BY_SUFFIX[suffix],
      isLive: false,
    }
  }

  const transcodedPath = subsonic.streamUrl(trackId, {
    format: 'mp3',
    maxBitRate: 320,
  })
  return {
    url: toAbsoluteUrl(transcodedPath, origin),
    contentType: 'audio/mpeg',
    isLive: false,
  }
}

export const buildCastQueueWindow = (
  queue = [],
  targetIndex = 0,
  maxItems = MAX_CAST_WINDOW_ITEMS,
) => {
  if (!Array.isArray(queue) || queue.length === 0) {
    return { items: [], startOffset: 0, windowStartIndex: 0, windowEndIndex: 0 }
  }

  const clampedTarget = Math.max(0, Math.min(targetIndex, queue.length - 1))
  if (queue.length <= maxItems) {
    return {
      items: queue,
      startOffset: clampedTarget,
      windowStartIndex: 0,
      windowEndIndex: queue.length,
    }
  }

  const lookbehind = Math.min(5, Math.floor(maxItems / 5))
  let windowStartIndex = Math.max(0, clampedTarget - lookbehind)
  let windowEndIndex = windowStartIndex + maxItems
  if (windowEndIndex > queue.length) {
    windowEndIndex = queue.length
    windowStartIndex = Math.max(0, windowEndIndex - maxItems)
  }

  return {
    items: queue.slice(windowStartIndex, windowEndIndex),
    startOffset: clampedTarget - windowStartIndex,
    windowStartIndex,
    windowEndIndex,
  }
}

/**
 * Plans incremental Chromecast queue synchronization so queuing operations
 * such as "Play Next" (PLAYER_PLAY_NEXT), "Add to Queue" (PLAYER_ADD_TRACKS),
 * deleting upcoming tracks, or reordering the queue drawer never restart or
 * interrupt the currently playing track.
 */
export const planCastQueueSync = (
  prevWindowUuids = [],
  nextQueue = [],
  {
    activeUuid = null,
    explicitPlayIndex = undefined,
    savedPlayIndex = 0,
    maxItems = MAX_CAST_WINDOW_ITEMS,
  } = {},
) => {
  if (!Array.isArray(nextQueue) || nextQueue.length === 0) {
    return { action: 'STOP' }
  }

  const fallbackIndex = Math.max(
    0,
    Math.min(savedPlayIndex >= 0 ? savedPlayIndex : 0, nextQueue.length - 1),
  )

  if (!Array.isArray(prevWindowUuids) || prevWindowUuids.length === 0) {
    const targetIndex =
      explicitPlayIndex != null
        ? Math.max(0, Math.min(explicitPlayIndex, nextQueue.length - 1))
        : fallbackIndex
    return { action: 'LOAD', targetIndex, startSeconds: 0 }
  }

  if (explicitPlayIndex != null) {
    const clampedExplicit = Math.max(
      0,
      Math.min(explicitPlayIndex, nextQueue.length - 1),
    )
    const requested = nextQueue[clampedExplicit]
    if (requested && requested.uuid !== activeUuid) {
      const nextAllUuids = nextQueue.map((item) => item.uuid)
      const sameWindow =
        nextAllUuids.length === prevWindowUuids.length &&
        nextAllUuids.every((u, i) => u === prevWindowUuids[i])
      if (sameWindow || prevWindowUuids.includes(requested.uuid)) {
        // Only JUMP if the rest of the queue was not replaced by a brand-new list
        const activeStillPresent =
          activeUuid && nextQueue.some((item) => item.uuid === activeUuid)
        if (sameWindow || activeStillPresent) {
          return {
            action: 'JUMP',
            targetIndex: clampedExplicit,
            targetUuid: requested.uuid,
            targetItem: requested,
          }
        }
      }
      return { action: 'LOAD', targetIndex: clampedExplicit, startSeconds: 0 }
    }
  }

  const activeIndex = activeUuid
    ? nextQueue.findIndex((item) => item.uuid === activeUuid)
    : -1

  if (activeIndex < 0) {
    const targetIndex =
      explicitPlayIndex != null
        ? Math.max(0, Math.min(explicitPlayIndex, nextQueue.length - 1))
        : fallbackIndex
    return { action: 'LOAD', targetIndex, startSeconds: 0 }
  }

  const maxAllowedWindow = maxItems * 2

  // Anchor diffing against the currently loaded receiver window so auto-advances
  // and incremental edits ("Play Next", "Add to Queue", delete, reorder) never
  // trigger unnecessary queue reloads, even on queues larger than 40 tracks.
  const firstIdx = nextQueue.findIndex(
    (item) => item.uuid === prevWindowUuids[0],
  )
  const isContiguousMatch =
    firstIdx >= 0 &&
    firstIdx + prevWindowUuids.length <= nextQueue.length &&
    prevWindowUuids.every((u, idx) => nextQueue[firstIdx + idx]?.uuid === u)

  if (isContiguousMatch && prevWindowUuids.includes(activeUuid)) {
    const currentEnd = firstIdx + prevWindowUuids.length
    if (currentEnd < nextQueue.length && prevWindowUuids.length < maxItems) {
      const newEnd = Math.min(nextQueue.length, firstIdx + maxItems)
      const insertedItems = nextQueue.slice(currentEnd, newEnd)
      const nextWindowItems = nextQueue.slice(firstIdx, newEnd)
      const nextWindowUuids = nextWindowItems.map((item) => item.uuid)
      return {
        action: 'INSERT',
        insertedItems,
        insertBeforeUuid: null,
        windowSpec: {
          items: nextWindowItems,
          windowStartIndex: firstIdx,
          windowEndIndex: newEnd,
          startOffset: activeIndex - firstIdx,
        },
        activeIndex,
        nextWindowUuids,
      }
    }

    const currentItems = nextQueue.slice(firstIdx, currentEnd)
    return {
      action: 'NONE',
      windowSpec: {
        items: currentItems,
        windowStartIndex: firstIdx,
        windowEndIndex: currentEnd,
        startOffset: activeIndex - firstIdx,
      },
      activeIndex,
      nextWindowUuids: prevWindowUuids,
    }
  }

  // Detect pure contiguous insertion inside (or prepended right before) the
  // loaded window, such as "Play Next" on both small and >40-track queues.
  const lastUuid = prevWindowUuids[prevWindowUuids.length - 1]
  const lastIdx = nextQueue.findIndex((item) => item.uuid === lastUuid)
  if (firstIdx >= 0 && lastIdx >= firstIdx) {
    const candidateStart =
      firstIdx > 0 && prevWindowUuids.length < maxItems ? 0 : firstIdx
    const candidateEnd = lastIdx + 1
    const candidateItems = nextQueue.slice(candidateStart, candidateEnd)
    const candidateUuids = candidateItems.map((item) => item.uuid)

    if (
      candidateUuids.length > prevWindowUuids.length &&
      candidateUuids.length <= maxAllowedWindow &&
      candidateUuids.includes(activeUuid)
    ) {
      let prefixLen = 0
      while (
        prefixLen < prevWindowUuids.length &&
        prevWindowUuids[prefixLen] === candidateUuids[prefixLen]
      ) {
        prefixLen += 1
      }

      let suffixLen = 0
      while (
        suffixLen < prevWindowUuids.length - prefixLen &&
        prevWindowUuids[prevWindowUuids.length - 1 - suffixLen] ===
          candidateUuids[candidateUuids.length - 1 - suffixLen]
      ) {
        suffixLen += 1
      }

      if (prefixLen + suffixLen === prevWindowUuids.length) {
        const insertEnd = candidateItems.length - suffixLen
        const insertedItems = candidateItems.slice(prefixLen, insertEnd)
        if (insertedItems.length > 0 && insertedItems.length <= maxItems) {
          const insertBeforeUuid =
            suffixLen > 0 ? prevWindowUuids[prefixLen] : null
          return {
            action: 'INSERT',
            insertedItems,
            insertBeforeUuid,
            windowSpec: {
              items: candidateItems,
              windowStartIndex: candidateStart,
              windowEndIndex: candidateEnd,
              startOffset: activeIndex - candidateStart,
            },
            activeIndex,
            nextWindowUuids: candidateUuids,
          }
        }
      }
    }
  }

  // Detect pure deletion of non-active tracks from the loaded receiver window
  const nextUuidToIdx = new Map(nextQueue.map((item, idx) => [item.uuid, idx]))
  const survivingUuids = prevWindowUuids.filter((u) => nextUuidToIdx.has(u))
  if (
    survivingUuids.length > 0 &&
    survivingUuids.length < prevWindowUuids.length &&
    survivingUuids.includes(activeUuid)
  ) {
    const survStart = nextUuidToIdx.get(survivingUuids[0])
    const survContiguous = survivingUuids.every(
      (u, idx) => nextQueue[survStart + idx]?.uuid === u,
    )
    if (survContiguous) {
      const keepSet = new Set(survivingUuids)
      const removedUuids = prevWindowUuids.filter((u) => !keepSet.has(u))
      const survItems = nextQueue.slice(
        survStart,
        survStart + survivingUuids.length,
      )
      return {
        action: 'REMOVE',
        removedUuids,
        windowSpec: {
          items: survItems,
          windowStartIndex: survStart,
          windowEndIndex: survStart + survivingUuids.length,
          startOffset: activeIndex - survStart,
        },
        activeIndex,
        nextWindowUuids: survivingUuids,
      }
    }
  }

  // Detect pure drag-and-drop reorder within the loaded receiver window
  const prevSet = new Set(prevWindowUuids)
  if (prevSet.has(activeUuid) && prevSet.size === prevWindowUuids.length) {
    let minIdx = Infinity
    let maxIdx = -1
    let allFound = true
    for (const u of prevWindowUuids) {
      const idx = nextUuidToIdx.get(u)
      if (idx == null) {
        allFound = false
        break
      }
      if (idx < minIdx) minIdx = idx
      if (idx > maxIdx) maxIdx = idx
    }
    if (allFound && maxIdx - minIdx + 1 === prevWindowUuids.length) {
      const reorderedItems = nextQueue.slice(minIdx, maxIdx + 1)
      const orderedUuids = reorderedItems.map((item) => item.uuid)
      return {
        action: 'REORDER',
        orderedUuids,
        windowSpec: {
          items: reorderedItems,
          windowStartIndex: minIdx,
          windowEndIndex: maxIdx + 1,
          startOffset: activeIndex - minIdx,
        },
        activeIndex,
        nextWindowUuids: orderedUuids,
      }
    }
  }

  const windowSpec = buildCastQueueWindow(nextQueue, activeIndex, maxItems)
  const nextWindowUuids = windowSpec.items.map((item) => item.uuid)

  // Fallback for complex windowed shifts: preserve active track at its current time
  return {
    action: 'RELOAD_PRESERVING_ACTIVE',
    windowSpec,
    activeIndex,
    nextWindowUuids,
  }
}

export const buildCastQueueItems = (chromeCastApi, items = [], origin) => {
  if (!chromeCastApi?.media) {
    return []
  }

  return items
    .map((item) => {
      const { url, contentType, isLive } = resolveCastStream(item, origin)
      if (!url) {
        return null
      }

      const song = item.song || {}
      const mediaInfo = new chromeCastApi.media.MediaInfo(url, contentType)
      mediaInfo.streamType = isLive
        ? chromeCastApi.media.StreamType.LIVE
        : chromeCastApi.media.StreamType.BUFFERED

      if (!isLive && (item.duration || song.duration)) {
        mediaInfo.duration = Number(item.duration || song.duration)
      }

      const metadata = new chromeCastApi.media.MusicTrackMediaMetadata()
      metadata.metadataType = chromeCastApi.media.MetadataType.MUSIC_TRACK
      metadata.title = song.title || item.name || ''
      metadata.artist = song.artist || item.singer || ''
      metadata.albumName = song.album || ''
      metadata.albumArtist = song.albumArtist || song.artist || ''
      if (song.trackNumber) {
        metadata.trackNumber = Number(song.trackNumber)
      }
      if (song.discNumber) {
        metadata.discNumber = Number(song.discNumber)
      }

      const coverUrl = toAbsoluteUrl(item.cover, origin)
      if (coverUrl && chromeCastApi.Image) {
        metadata.images = [new chromeCastApi.Image(coverUrl)]
      }

      mediaInfo.metadata = metadata
      mediaInfo.customData = {
        trackId: item.trackId,
        uuid: item.uuid,
        isRadio: Boolean(item.isRadio),
        albumId: song.albumId || '',
      }

      const queueItem = new chromeCastApi.media.QueueItem(mediaInfo)
      queueItem.autoplay = true
      if (!isLive) {
        queueItem.preloadTime = 10
      }
      return queueItem
    })
    .filter(Boolean)
}

export const formatCastTime = (seconds) => {
  if (!Number.isFinite(seconds) || seconds < 0) {
    return '0:00'
  }
  const total = Math.floor(seconds)
  const hrs = Math.floor(total / 3600)
  const mins = Math.floor((total % 3600) / 60)
  const secs = total % 60
  const paddedSecs = secs < 10 ? `0${secs}` : `${secs}`
  if (hrs > 0) {
    const paddedMins = mins < 10 ? `0${mins}` : `${mins}`
    return `${hrs}:${paddedMins}:${paddedSecs}`
  }
  return `${mins}:${paddedSecs}`
}
