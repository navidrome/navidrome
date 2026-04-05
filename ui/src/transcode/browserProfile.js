// Each entry: { codec name for the server, container, mime: [MIME probe strings] }
export const CODEC_PROBES = [
  { codec: 'mp3', container: 'mp3', mime: ['audio/mpeg; codecs="mp3"'] },
  { codec: 'opus', container: 'ogg', mime: ['audio/ogg; codecs="opus"'] },
  { codec: 'vorbis', container: 'ogg', mime: ['audio/ogg; codecs="vorbis"'] },
  {
    codec: 'flac',
    container: 'flac',
    mime: ['audio/flac', 'audio/flac; codecs="flac"'],
  },
  { codec: 'wav', container: 'wav', mime: ['audio/wav; codecs="1"'] },
  { codec: 'alac', container: 'mp4', mime: ['audio/mp4; codecs="alac"'] },
  { codec: 'aac', container: 'mp4', mime: ['audio/mp4; codecs="mp4a.40.2"'] },
]

// Transcoding targets in preference order (lossless first, then lossy).
// Derived from CODEC_PROBES to avoid duplicating MIME strings.
// MP3 is always included as a universal fallback.
const TRANSCODE_CODECS = ['flac', 'opus', 'mp3']

// Safari transcoding is limited to mp3 only. Safari cannot reliably stream
// Ogg containers (reports canPlayType support but fails on non-seekable
// transcoded streams), and FLAC transcoding also fails in practice.
const SAFARI_TRANSCODE_CODECS = ['mp3']

function canPlay(audio, mimeList) {
  return mimeList.some((m) => {
    const result = audio.canPlayType(m)
    return result === 'probably' || result === 'maybe'
  })
}

function probeSupported(audio, probes) {
  return probes.filter(({ mime }) => canPlay(audio, mime))
}

function isSafari() {
  const ua = navigator.userAgent
  return (
    ua.includes('Safari') && !ua.includes('Chrome') && !ua.includes('Chromium')
  )
}

export function detectBrowserProfile() {
  const audio = new Audio()

  const directPlayProfiles = probeSupported(audio, CODEC_PROBES).map(
    ({ codec, container }) => ({
      containers: [container],
      audioCodecs: [codec],
      protocols: ['http'],
    }),
  )

  // Build transcoding profiles from supported codecs, always keeping mp3 as fallback.
  // Safari is limited to mp3 transcoding only.
  const transcodeCodecs = isSafari()
    ? SAFARI_TRANSCODE_CODECS
    : TRANSCODE_CODECS
  const transcodingProfiles = transcodeCodecs.reduce((profiles, codec) => {
    const probe = CODEC_PROBES.find((p) => p.codec === codec)
    if (!probe) return profiles
    if (canPlay(audio, probe.mime) || codec === 'mp3') {
      profiles.push({
        container: probe.container,
        audioCodec: codec,
        protocol: 'http',
      })
    }
    return profiles
  }, [])

  return {
    name: 'NavidromeUI',
    platform: navigator.userAgent,
    directPlayProfiles,
    transcodingProfiles,
    codecProfiles: [],
  }
}
