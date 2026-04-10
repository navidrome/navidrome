import React from 'react'
import PropTypes from 'prop-types'
import TagEditorDialog from './TagEditorDialog'

const songFields = [
  { name: 'title', label: 'Title' },
  {
    name: 'artist',
    label: 'Artist(s)',
    helperText: 'Separate multiple values with semicolons',
  },
  { name: 'album', label: 'Album' },
  {
    name: 'albumArtist',
    label: 'Album artist(s)',
    helperText: 'Separate multiple values with semicolons',
  },
  { name: 'trackNumber', label: 'Track number' },
  { name: 'discNumber', label: 'Disc number' },
  { name: 'date', label: 'Recording date / year' },
  { name: 'releaseDate', label: 'Release date' },
  { name: 'originalDate', label: 'Original date' },
  {
    name: 'genre',
    label: 'Genre(s)',
    helperText: 'Separate multiple values with semicolons',
  },
  { name: 'comment', label: 'Comment', multiline: true, rows: 3 },
  { name: 'lyrics', label: 'Lyrics', multiline: true, rows: 6 },
]

const TagEditorSongDialog = ({ songId, open, onClose }) => {
  if (!songId) {
    return null
  }

  return (
    <TagEditorDialog
      open={open}
      onClose={onClose}
      title="Edit song tags"
      endpoint={`/tag-editor/song/${songId}`}
      fields={songFields}
      saveLabel="Save song tags"
    />
  )
}

TagEditorSongDialog.propTypes = {
  onClose: PropTypes.func.isRequired,
  open: PropTypes.bool,
  songId: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
}

TagEditorSongDialog.defaultProps = {
  open: false,
  songId: null,
}

export default TagEditorSongDialog
