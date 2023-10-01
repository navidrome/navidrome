import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { Button, useTranslate, useUnselectAll } from 'react-admin'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { openDeleteMenu } from '../actions'

export const DeleteButton = ({ resource, selectedIds, className }) => {
  const translate = useTranslate()
  const dispatch = useDispatch()

  const handleClick = () => {
    dispatch(openDeleteMenu(selectedIds))
  }

  return (
    <Button
      aria-controls="simple-menu"
      aria-haspopup="true"
      onClick={handleClick}
      className={className}
      label={translate('resources.song.actions.delete')}
    >
      <PlaylistAddIcon />
    </Button>
  )
}

DeleteButton.propTypes = {
  resource: PropTypes.string.isRequired,
  selectedIds: PropTypes.arrayOf(PropTypes.string).isRequired,
}
