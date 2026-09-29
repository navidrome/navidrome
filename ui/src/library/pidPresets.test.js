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
  it('is false when nothing changed', () => {
    expect(
      pidConfigChanged({ pidAlbum: 'folder', pidTrack: '' }, record, globals),
    ).toBe(false)
  })
  it('treats missing and empty values as the same', () => {
    expect(pidConfigChanged({ pidAlbum: 'folder' }, record, globals)).toBe(
      false,
    )
  })
  it('is true when the album PID changed', () => {
    expect(
      pidConfigChanged({ pidAlbum: '', pidTrack: '' }, record, globals),
    ).toBe(true)
  })
  it('is true when the track PID changed', () => {
    expect(
      pidConfigChanged(
        { pidAlbum: 'folder', pidTrack: 'title' },
        record,
        globals,
      ),
    ).toBe(true)
  })
  it('is false when Custom is set to the global value', () => {
    expect(
      pidConfigChanged(
        { pidAlbum: 'folder', pidTrack: globals.pidTrack },
        record,
        globals,
      ),
    ).toBe(false)
  })
  it('ignores case-only changes', () => {
    expect(
      pidConfigChanged({ pidAlbum: 'FOLDER', pidTrack: '' }, record, globals),
    ).toBe(false)
  })
  it('ignores whitespace-only changes', () => {
    expect(
      pidConfigChanged(
        { pidAlbum: ' folder ', pidTrack: '  ' },
        record,
        globals,
      ),
    ).toBe(false)
  })
  it('is true when an empty value becomes a custom one that is not the global', () => {
    expect(
      pidConfigChanged(
        { pidAlbum: 'folder', pidTrack: 'title' },
        { pidAlbum: 'folder', pidTrack: '' },
        globals,
      ),
    ).toBe(true)
  })
})
