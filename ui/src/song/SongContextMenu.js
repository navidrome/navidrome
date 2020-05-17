import React, { useState } from 'react'
import { useDispatch } from 'react-redux'
import { useTranslate } from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { addTracks, setTrack } from '../audioplayer'
import { AddToPlaylistMenu } from '../common'
import NestedMenuItem from 'material-ui-nested-menu-item'

export const SongContextMenu = ({ record }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)
  const options = {
    playNow: {
      label: translate('resources.song.actions.playNow'),
      action: (record) => setTrack(record),
    },
    addToQueue: {
      label: translate('resources.song.actions.addToQueue'),
      action: (record) => addTracks({ [record.id]: record }),
    },
  }

  const handleClick = (e) => {
    setAnchorEl(e.currentTarget)
    e.stopPropagation()
  }

  const handleClose = (e) => {
    setAnchorEl(null)
    e.stopPropagation()
  }

  const handleItemClick = (e) => {
    e.preventDefault()
    setAnchorEl(null)
    const key = e.target.getAttribute('value')
    dispatch(options[key].action(record))
    e.stopPropagation()
  }

  const open = Boolean(anchorEl)

  return (
    <>
      <IconButton onClick={handleClick} size={'small'}>
        <MoreVertIcon />
      </IconButton>
      <Menu
        id={'menu' + record.id}
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
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
            selectedIds={[record.id]}
            onClose={() => setAnchorEl(null)}
          />
        </NestedMenuItem>
      </Menu>
    </>
  )
}
