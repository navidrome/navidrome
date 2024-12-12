import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import { Button, useTranslate, useUnselectAll } from 'react-admin'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { openAddToPlaylist } from '../actions'
import { Add } from '@material-ui/icons'
import { IconButton } from '@material-ui/core'

/**
 * @component
 * @param {{
 *  selectedIds: string[],
 *  resource?: string,
 *  className?: string,
 *  compact?: boolean
 *  disabled?: boolean
 * }}
 */
export const AddToPlaylistButton = ({
  resource,
  selectedIds,
  className,
  compact,
  disabled,
}) => {
  const translate = useTranslate()
  const dispatch = useDispatch()
  const unselectAll = useUnselectAll()

  const handleClick = () => {
    dispatch(
      openAddToPlaylist({
        selectedIds,
        onSuccess: () => unselectAll(resource),
      }),
    )
  }

  if (compact) {
    return (
      <IconButton
        aria-controls="simple-menu"
        aria-haspopup="true"
        size={'small'}
        onClick={handleClick}
        className={className}
        label={translate('resources.song.actions.addToPlaylist')}
        disabled={disabled}
      >
        <Add fontSize="small" />
      </IconButton>
    )
  }

  return (
    <Button
      aria-controls="simple-menu"
      aria-haspopup="true"
      onClick={handleClick}
      className={className}
      label={translate('resources.song.actions.addToPlaylist')}
      disabled={disabled}
    >
      <PlaylistAddIcon />
    </Button>
  )
}

AddToPlaylistButton.propTypes = {
  resource: PropTypes.string,
  selectedIds: PropTypes.arrayOf(PropTypes.string).isRequired,
  className: PropTypes.string,
  compact: PropTypes.bool,
  disabled: PropTypes.bool,
}
