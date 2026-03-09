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

// Default transcoding targets — ordered by preference.
// These are attempted if direct play is not possible.
const DEFAULT_TRANSCODING_PROFILES = [
  { container: 'ogg', audioCodec: 'opus', protocol: 'http' },
  { container: 'mp3', audioCodec: 'mp3', protocol: 'http' },
]

export function detectBrowserProfile() {
  const audio = new Audio()
  const directPlayProfiles = []

  for (const { codec, container, mime } of CODEC_PROBES) {
    if (audio.canPlayType(mime) === 'probably') {
      directPlayProfiles.push({
        containers: [container],
        audioCodecs: [codec],
        protocols: ['http'],
      })
    }
  }

  return {
    name: 'NavidromeUI',
    platform: navigator.userAgent,
    directPlayProfiles,
    transcodingProfiles: DEFAULT_TRANSCODING_PROFILES,
    codecProfiles: [],
  }
}
