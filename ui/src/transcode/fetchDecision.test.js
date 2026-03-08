import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'

// Mock httpClient before importing module under test
vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
}))

import { fetchTranscodeDecision } from './fetchDecision'
import { httpClient } from '../dataProvider'

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

  const fakeJson = {
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

    httpClient.mockResolvedValue({ json: fakeJson })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('makes a POST request to getTranscodeDecision with correct URL', async () => {
    await fetchTranscodeDecision('song-1', fakeProfile)

    expect(httpClient).toHaveBeenCalledTimes(1)
    const [url, options] = httpClient.mock.calls[0]
    expect(url).toContain('getTranscodeDecision')
    expect(url).toContain('mediaId=song-1')
    expect(url).toContain('mediaType=song')
    expect(options.method).toBe('POST')
  })

  it('sends the browser profile as JSON body', async () => {
    await fetchTranscodeDecision('song-1', fakeProfile)

    const [, options] = httpClient.mock.calls[0]
    expect(JSON.parse(options.body)).toEqual(fakeProfile)
  })

  it('returns the transcodeDecision from response', async () => {
    const result = await fetchTranscodeDecision('song-1', fakeProfile)
    expect(result).toEqual(fakeJson['subsonic-response'].transcodeDecision)
  })

  it('throws on HTTP error (httpClient rejects)', async () => {
    httpClient.mockRejectedValue(new Error('Server Error'))

    await expect(
      fetchTranscodeDecision('song-1', fakeProfile),
    ).rejects.toThrow()
  })

  it('throws on Subsonic error response', async () => {
    httpClient.mockResolvedValue({
      json: {
        'subsonic-response': {
          status: 'failed',
          error: { code: 70, message: 'not found' },
        },
      },
    })

    await expect(
      fetchTranscodeDecision('song-1', fakeProfile),
    ).rejects.toThrow()
  })
})
