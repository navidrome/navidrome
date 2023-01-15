import React, { useState } from 'react'
import { IconButton, Menu, MenuItem } from '@material-ui/core'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import PropTypes from 'prop-types'
import { useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'

import { setTrack } from '../actions'
import { songFromRadio } from './helper'

export const RadioContextMenu = ({ record, className }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [anchorEl, setAnchorEl] = useState(null)

  const newRecord = songFromRadio(record)

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
    options[key].action(newRecord)
    e.stopPropagation()
  }

  const options = {
    playNow: {
      enabled: true,
      label: translate('resources.radio.actions.playNow'),
      action: (record) => dispatch(setTrack(record)),
    },
  }

  const open = Boolean(anchorEl)

  return (
    <span className={className}>
      <IconButton onClick={handleClick} size="small">
        <MoreVertIcon fontSize="small" />
      </IconButton>
      <Menu
        id={'menu' + newRecord.id}
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
      >
        {Object.keys(options).map(
          (key) =>
            options[key].enabled && (
              <MenuItem value={key} key={key} onClick={handleItemClick}>
                {options[key].label}
              </MenuItem>
            )
        )}
      </Menu>
    </span>
  )
}

RadioContextMenu.propTypes = {
  record: PropTypes.object.isRequired,
}

RadioContextMenu.defaultProps = {
  record: {},
}
