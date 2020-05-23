import React, { useState } from 'react'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { addTracks, playTracks, shuffleTracks } from '../audioplayer'
import NestedMenuItem from 'material-ui-nested-menu-item'
import { AddToPlaylistMenu } from '../common'

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
  const notify = useNotify()
  const [anchorEl, setAnchorEl] = useState(null)
  const open = Boolean(anchorEl)
  const options = {
    play: {
      label: translate('resources.album.actions.playAll'),
      action: (data) => playTracks(data),
    },
    addToQueue: {
      label: translate('resources.album.actions.addToQueue'),
      action: (data) => addTracks(data),
    },
    shuffle: {
      label: translate('resources.album.actions.shuffle'),
      action: (data) => shuffleTracks(data),
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
        dispatch(options[key].action(adata))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
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
        <NestedMenuItem
          label={translate('resources.song.actions.addToPlaylist')}
          parentMenuOpen={open}
        >
          <AddToPlaylistMenu
            albumId={record.id}
            onClose={() => setAnchorEl(null)}
          />
        </NestedMenuItem>
      </Menu>
    </div>
  )
}
export default AlbumContextMenu
