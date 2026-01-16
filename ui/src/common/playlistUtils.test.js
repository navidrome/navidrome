import {
  isWritable,
  isReadOnly,
  isSmartPlaylist,
  isGlobalPlaylist,
  canChangeTracks,
} from './playlistUtils'

describe('playlistUtils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('isWritable', () => {
    it('returns true if user is the owner', () => {
      localStorage.setItem('userId', 'user1')
      expect(isWritable('user1')).toBe(true)
    })

    it('returns true if user is an admin', () => {
      localStorage.setItem('role', 'admin')
      expect(isWritable('user1')).toBe(true)
    })

    it('returns false if user is not the owner and not an admin', () => {
      localStorage.setItem('userId', 'user2')
      expect(isWritable('user1')).toBe(false)
    })
  })

  describe('isReadOnly', () => {
    it('returns true if user is not the owner and not an admin', () => {
      localStorage.setItem('userId', 'user2')
      expect(isReadOnly('user1')).toBe(true)
    })

    it('returns false if user is the owner', () => {
      localStorage.setItem('userId', 'user1')
      expect(isReadOnly('user1')).toBe(false)
    })

    it('returns false if user is an admin', () => {
      localStorage.setItem('role', 'admin')
      expect(isReadOnly('user1')).toBe(false)
    })
  })

  describe('isSmartPlaylist', () => {
    it('returns true if playlist has rules', () => {
      const playlist = { rules: [] }
      expect(isSmartPlaylist(playlist)).toBe(true)
    })

    it('returns false if playlist does not have rules', () => {
      const playlist = {}
      expect(isSmartPlaylist(playlist)).toBe(false)
    })
  })

  describe('isGlobalPlaylist', () => {
    it('returns true if playlist is smart and global', () => {
      const playlist = { rules: [], global: true }
      expect(isGlobalPlaylist(playlist)).toBe(true)
    })

    it('returns false if playlist is smart but not global', () => {
      const playlist = { rules: [], global: false }
      expect(isGlobalPlaylist(playlist)).toBe(false)
    })

    it('returns false if playlist is not smart even if global is true', () => {
      const playlist = { global: true }
      expect(isGlobalPlaylist(playlist)).toBe(false)
    })

    it('returns false if playlist is not smart and not global', () => {
      const playlist = {}
      expect(isGlobalPlaylist(playlist)).toBe(false)
    })
  })

  describe('canChangeTracks', () => {
    it('returns true if user is the owner and playlist is not smart', () => {
      localStorage.setItem('userId', 'user1')
      const playlist = { ownerId: 'user1' }
      expect(canChangeTracks(playlist)).toBe(true)
    })

    it('returns false if user is not the owner', () => {
      localStorage.setItem('userId', 'user2')
      const playlist = { ownerId: 'user1' }
      expect(canChangeTracks(playlist)).toBe(false)
    })

    it('returns false if playlist is smart', () => {
      localStorage.setItem('userId', 'user1')
      const playlist = { ownerId: 'user1', rules: [] }
      expect(canChangeTracks(playlist)).toBe(false)
    })
  })
})
