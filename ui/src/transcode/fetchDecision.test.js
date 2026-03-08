import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { fetchTranscodeDecision } from './fetchDecision'

describe('fetchTranscodeDecision', () => {
  const fakeProfile = {
    name: 'NavidromeUI',
    platform: 'test',
    directPlayProfiles: [
      { containers: ['mp3'], audioCodecs: ['mp3'], protocols: ['http'] },
    ],
    transcodingProfiles: [],
    codecProfiles: [],
  }

  const fakeResponse = {
    'subsonic-response': {
      status: 'ok',
      transcodeDecision: {
        canDirectPlay: true,
        canTranscode: false,
        transcodeParams: 'jwt-token',
        sourceStream: { codec: 'mp3' },
      },
    },
  }

  beforeEach(() => {
    localStorage.setItem('username', 'testuser')
    localStorage.setItem('subsonic-token', 'testtoken')
    localStorage.setItem('subsonic-salt', 'testsalt')

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(fakeResponse),
      }),
    )
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('makes a POST request to getTranscodeDecision with correct URL', async () => {
    await fetchTranscodeDecision('song-1', fakeProfile)

    expect(fetch).toHaveBeenCalledTimes(1)
    const [url, options] = fetch.mock.calls[0]
    expect(url).toContain('getTranscodeDecision')
    expect(url).toContain('mediaId=song-1')
    expect(url).toContain('mediaType=song')
    expect(options.method).toBe('POST')
  })

  it('sends the browser profile as JSON body', async () => {
    await fetchTranscodeDecision('song-1', fakeProfile)

    const [, options] = fetch.mock.calls[0]
    expect(options.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(options.body)).toEqual(fakeProfile)
  })

  it('returns the transcodeDecision from response', async () => {
    const result = await fetchTranscodeDecision('song-1', fakeProfile)
    expect(result).toEqual(fakeResponse['subsonic-response'].transcodeDecision)
  })

  it('throws on non-ok HTTP response', async () => {
    fetch.mockResolvedValue({ ok: false, status: 500, statusText: 'Server Error' })

    await expect(
      fetchTranscodeDecision('song-1', fakeProfile),
    ).rejects.toThrow()
  })

  it('throws on Subsonic error response', async () => {
    fetch.mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          'subsonic-response': {
            status: 'failed',
            error: { code: 70, message: 'not found' },
          },
        }),
    })

    await expect(
      fetchTranscodeDecision('song-1', fakeProfile),
    ).rejects.toThrow()
  })
})
