import React from 'react'
import PropTypes from 'prop-types'
import { useGetList, useTranslate } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import Avatar from '@material-ui/core/Avatar'
import List from '@material-ui/core/List'
import ListItem from '@material-ui/core/ListItem'
import ListItemAvatar from '@material-ui/core/ListItemAvatar'
import ListItemText from '@material-ui/core/ListItemText'
import DialogTitle from '@material-ui/core/DialogTitle'
import Dialog from '@material-ui/core/Dialog'
import { blue } from '@material-ui/core/colors'
import PlaylistIcon from '../icons/Playlist'

const useStyles = makeStyles({
  avatar: {
    backgroundColor: blue[100],
    color: blue[600],
  },
})

function SelectPlaylistDialog(props) {
  const classes = useStyles()
  const translate = useTranslate()
  const { onClose, selectedValue, open } = props
  const { ids, data, loaded } = useGetList(
    'playlist',
    { page: 1, perPage: -1 },
    { field: '', order: '' },
    {}
  )

  if (!loaded) {
    return <div />
  }

  const handleClose = () => {
    onClose(selectedValue)
  }

  const handleListItemClick = (value) => {
    onClose(value)
  }

  return (
    <Dialog
      onClose={handleClose}
      aria-labelledby="select-playlist-dialog-title"
      open={open}
      scroll={'paper'}
    >
      <DialogTitle id="select-playlist-dialog-title">
        {translate('resources.playlist.actions.selectPlaylist')}
      </DialogTitle>
      <List>
        {ids.map((id) => (
          <ListItem button onClick={() => handleListItemClick(id)} key={id}>
            <ListItemAvatar>
              <Avatar className={classes.avatar}>
                <PlaylistIcon />
              </Avatar>
            </ListItemAvatar>
            <ListItemText primary={data[id].name} />
          </ListItem>
        ))}
      </List>
    </Dialog>
  )
}

SelectPlaylistDialog.propTypes = {
  onClose: PropTypes.func.isRequired,
  open: PropTypes.bool.isRequired,
  selectedValue: PropTypes.string.isRequired,
}

export default SelectPlaylistDialog
