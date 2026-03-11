// Each entry: { codec name for the server, container, MIME to probe }
export const CODEC_PROBES = [
  { codec: 'mp3', container: 'mp3', mime: 'audio/mpeg' },
  { codec: 'aac', container: 'mp4', mime: 'audio/mp4; codecs="mp4a.40.2"' },
  { codec: 'opus', container: 'ogg', mime: 'audio/ogg; codecs="opus"' },
  { codec: 'vorbis', container: 'ogg', mime: 'audio/ogg; codecs="vorbis"' },
  { codec: 'flac', container: 'flac', mime: 'audio/flac' },
  { codec: 'wav', container: 'wav', mime: 'audio/wav' },
  { codec: 'alac', container: 'mp4', mime: 'audio/mp4; codecs="alac"' },
]

// Transcoding targets in preference order (lossless first, then lossy).
// Derived from CODEC_PROBES to avoid duplicating MIME strings.
// MP3 is always included as a universal fallback.
const TRANSCODE_CODECS = ['flac', 'opus', 'mp3']

function probeSupported(audio, probes) {
  return probes.filter(({ mime }) => audio.canPlayType(mime) === 'probably')
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

  // Build transcoding profiles from supported codecs, always keeping mp3 as fallback
  const transcodingProfiles = TRANSCODE_CODECS.reduce((profiles, codec) => {
    const probe = CODEC_PROBES.find((p) => p.codec === codec)
    if (audio.canPlayType(probe.mime) === 'probably' || codec === 'mp3') {
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
