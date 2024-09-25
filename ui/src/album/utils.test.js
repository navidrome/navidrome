import { removeAlbumCommentsFromSongs } from './utils.js'

describe('removeAlbumCommentsFromSongs', () => {
  const data = { 1: { comment: 'one' }, 2: { comment: 'two' } }
  it('does not remove song comments if album does not have comment', () => {
    const album = { comment: '' }
    removeAlbumCommentsFromSongs({ album, data })
    expect(data['1'].comment).toEqual('one')
    expect(data['2'].comment).toEqual('two')
  })

  it('removes song comments if album has comment', () => {
    const album = { comment: 'test' }
    removeAlbumCommentsFromSongs({ album, data })
    expect(data['1'].comment).toEqual('')
    expect(data['2'].comment).toEqual('')
  })

  it('does not crash if album and data arr not available', () => {
    expect(() => {
      removeAlbumCommentsFromSongs({})
    }).not.toThrow()
  })
})
