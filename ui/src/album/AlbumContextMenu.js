import React, { useState } from 'react'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { playAlbum, shuffleAlbum } from '../audioplayer'

const useStyles = makeStyles({
  icon: {
    color: (props) => props.color,
  },
})

const AlbumContextMenu = ({ record, color }) => {
  const classes = useStyles({ color })
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
    e.stopPropagation()
  }

  const handleOnClose = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    e.stopPropagation()
  }

  const handleItemClick = (e) => {
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    dataProvider
      .getList('albumSong', {
        pagination: { page: 1, perPage: -1 },
        sort: { field: 'trackNumber', order: 'ASC' },
        filter: { album_id: record.id },
      })
      .then((response) => {
        const adata = response.data.reduce(
          (acc, cur) => ({ ...acc, [cur.id]: cur }),
          {}
        )
        dispatch(options[key].action(adata, response.data[0].id))
      })
    e.stopPropagation()
  }

  return (
    <div>
      <IconButton
        aria-label="more"
        aria-controls="context-menu"
        aria-haspopup="true"
        className={classes.icon}
        onClick={handleClick}
        size={'small'}
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
