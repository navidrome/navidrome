import React, { useState } from 'react'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { useDataProvider, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { playAlbum, shuffleAlbum } from '../audioplayer'

const AlbumContextMenu = (props) => {
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  const options = {
    play: {
      label: translate('resources.album.actions.playAll'),
      action: (data, id) => playAlbum(id, data),
    },
    shuffle: {
      label: translate('resources.album.actions.shuffle'),
      action: (data) => shuffleAlbum(data),
    },
  }

  const handleClick = (e) => {
    e.preventDefault()
    setAnchorEl(e.currentTarget)
  }

  const handleOnClose = (e) => {
    e.preventDefault()
    setAnchorEl(null)
  }

  const handleItemClick = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    dataProvider
      .getList('albumSong', {
        pagination: { page: 0, perPage: 1000 },
        sort: { field: 'trackNumber', order: 'ASC' },
        filter: { album_id: props.id },
      })
      .then((response) => {
        const adata = response.data.reduce(
          (acc, cur) => ({ ...acc, [cur.id]: cur }),
          {}
        )
        dispatch(options[key].action(adata, response.data[0].id))
      })
  }

  return (
    <div>
      <IconButton
        aria-label="more"
        aria-controls="context-menu"
        aria-haspopup="true"
        onClick={handleClick}
      >
        <MoreVertIcon />
      </IconButton>
      <Menu
        id="context-menu"
        anchorEl={anchorEl}
        keepMounted
        open={open}
        onClose={handleOnClose}
      >
        {Object.keys(options).map((key) => (
          <MenuItem value={key} key={key} onClick={handleItemClick}>
            {options[key].label}
          </MenuItem>
        ))}
      </Menu>
    </div>
  )
}
export default AlbumContextMenu
