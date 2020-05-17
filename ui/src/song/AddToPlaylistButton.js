import React from 'react'
import { Button, useTranslate, useUnselectAll } from 'react-admin'
import { Menu } from '@material-ui/core'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { AddToPlaylistMenu } from '../common'

const AddToPlaylistButton = ({ resource, selectedIds }) => {
  const [anchorEl, setAnchorEl] = React.useState(null)
  const translate = useTranslate()
  const unselectAll = useUnselectAll()

  const handleClick = (event) => {
    setAnchorEl(event.currentTarget)
  }

  const handleClose = () => {
    setAnchorEl(null)
    unselectAll(resource)
  }

  return (
    <>
      <Button
        aria-controls="simple-menu"
        aria-haspopup="true"
        onClick={handleClick}
        color="secondary"
        label={translate('resources.song.actions.addToPlaylist')}
      >
        <PlaylistAddIcon />
      </Button>
      <Menu
        id="simple-menu"
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        <AddToPlaylistMenu
          selectedIds={selectedIds}
          menuOpen={Boolean(anchorEl)}
        />
      </Menu>
    </>
  )
}

export default AddToPlaylistButton
