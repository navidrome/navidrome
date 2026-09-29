import { describe, it, expect } from 'vitest'
import {
  PID_CUSTOM,
  PID_FOLDER,
  PID_GLOBAL,
  pidConfigChanged,
  pidModeFromValue,
  pidValueForMode,
} from './pidPresets'

describe('pidModeFromValue', () => {
  it('maps an empty value to the global setting', () => {
    expect(pidModeFromValue('', true)).toBe(PID_GLOBAL)
    expect(pidModeFromValue(undefined, true)).toBe(PID_GLOBAL)
  })
  it('maps folder to the Folder preset when allowed', () => {
    expect(pidModeFromValue('folder', true)).toBe(PID_FOLDER)
  })
  it('maps folder to Custom when the Folder preset is not offered', () => {
    expect(pidModeFromValue('folder', false)).toBe(PID_CUSTOM)
  })
  it('maps any other value to Custom', () => {
    expect(pidModeFromValue('album|title', true)).toBe(PID_CUSTOM)
  })
})

describe('pidValueForMode', () => {
  it('stores an empty value for the global setting', () => {
    expect(pidValueForMode(PID_GLOBAL, 'album')).toBe('')
  })
  it('stores folder for the Folder preset', () => {
    expect(pidValueForMode(PID_FOLDER, '')).toBe('folder')
  })
  it('keeps the current value for Custom', () => {
    expect(pidValueForMode(PID_CUSTOM, 'album|title')).toBe('album|title')
    expect(pidValueForMode(PID_CUSTOM, undefined)).toBe('')
  })
})

describe('pidConfigChanged', () => {
  const record = { pidAlbum: 'folder', pidTrack: '' }
  const globals = {
    pidAlbum: 'musicbrainz_albumid|albumartistid,album',
    pidTrack: 'musicbrainz_trackid|albumid,discnumber,tracknumber,title',
  }
  it.each([
    ['nothing changed', { pidAlbum: 'folder', pidTrack: '' }, false],
    ['a missing value equals an empty one', { pidAlbum: 'folder' }, false],
    [
      'Custom set to the global value',
      { pidAlbum: 'folder', pidTrack: globals.pidTrack },
      false,
    ],
    ['a case-only change', { pidAlbum: 'FOLDER', pidTrack: '' }, false],
    [
      'a whitespace-only change',
      { pidAlbum: ' folder ', pidTrack: '  ' },
      false,
    ],
    ['the album PID changed', { pidAlbum: '', pidTrack: '' }, true],
    ['the track PID changed', { pidAlbum: 'folder', pidTrack: 'title' }, true],
  ])('%s', (_, values, expected) => {
    expect(pidConfigChanged(values, record, globals)).toBe(expected)
  })
})
