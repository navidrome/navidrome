import React from 'react'
import PropTypes from 'prop-types'
import { Button } from 'react-admin'
import EditIcon from '@material-ui/icons/Edit'
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

const TagEditorSongButton = ({ songId }) => {
  const [open, setOpen] = React.useState(false)

  if (!songId || localStorage.getItem('role') !== 'admin') {
    return null
  }

  return (
    <>
      <Button label="Edit tags" onClick={() => setOpen(true)}>
        <EditIcon />
      </Button>
      <TagEditorDialog
        open={open}
        onClose={() => setOpen(false)}
        title="Edit song tags"
        endpoint={`/tag-editor/song/${songId}`}
        fields={songFields}
        saveLabel="Save song tags"
      />
    </>
  )
}

TagEditorSongButton.propTypes = {
  songId: PropTypes.oneOfType([PropTypes.number, PropTypes.string]).isRequired,
}

export default TagEditorSongButton
