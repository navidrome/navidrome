import React, { useState } from 'react'
import { useDispatch } from 'react-redux'
import { useTranslate } from 'react-admin'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { addTrack, setTrack } from '../audioplayer'

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
      action: (record) => addTrack(record),
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

  return (
    <>
      <IconButton onClick={handleClick}>
        <MoreVertIcon />
      </IconButton>
      <Menu
        id={'menu' + record.id}
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {Object.keys(options).map((key) => (
          <MenuItem value={key} key={key} onClick={handleItemClick}>
            {options[key].label}
          </MenuItem>
        ))}
      </Menu>
    </>
  )
}
