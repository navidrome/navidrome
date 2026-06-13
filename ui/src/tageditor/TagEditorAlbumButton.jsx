import React from 'react'
import PropTypes from 'prop-types'
import { Button } from 'react-admin'
import EditIcon from '@material-ui/icons/Edit'
import TagEditorDialog from './TagEditorDialog'

const albumFields = [
  { name: 'name', label: 'Album title' },
  {
    name: 'albumArtist',
    label: 'Album artist(s)',
    helperText: 'Separate multiple values with semicolons',
  },
  { name: 'date', label: 'Recording date / year' },
  { name: 'releaseDate', label: 'Release date' },
  { name: 'originalDate', label: 'Original date' },
  {
    name: 'genre',
    label: 'Genre(s)',
    helperText: 'Separate multiple values with semicolons',
  },
  { name: 'comment', label: 'Comment', multiline: true, rows: 4 },
  { name: 'compilation', label: 'Compilation', type: 'boolean' },
]

const TagEditorAlbumButton = ({ albumId }) => {
  const [open, setOpen] = React.useState(false)

  if (!albumId || localStorage.getItem('role') !== 'admin') {
    return null
  }

  return (
    <>
      <Button label="Edit album tags" onClick={() => setOpen(true)}>
        <EditIcon />
      </Button>
      <TagEditorDialog
        open={open}
        onClose={() => setOpen(false)}
        title="Edit album tags"
        endpoint={`/tag-editor/album/${albumId}`}
        fields={albumFields}
        saveLabel="Save album tags"
        onSaved={(data) => {
          if (data?.id && data.id !== albumId) {
            window.location.hash = `#/album/${data.id}/show`
          }
        }}
      />
    </>
  )
}

TagEditorAlbumButton.propTypes = {
  albumId: PropTypes.oneOfType([PropTypes.number, PropTypes.string]).isRequired,
}

export default TagEditorAlbumButton
