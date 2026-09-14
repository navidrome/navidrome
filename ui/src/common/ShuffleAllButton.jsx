import React, { useState } from 'react'
import { Button, useDataProvider, useNotify, useTranslate } from 'react-admin'
import { useDispatch } from 'react-redux'
import { IconButton, Tooltip } from '@material-ui/core'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import { shuffleTracks } from '../actions'
import PropTypes from 'prop-types'

const heroStyle = {
  minHeight: 72,
  width: '100%',
  fontSize: '1.25rem',
  fontWeight: 600,
  textTransform: 'none',
  borderRadius: 8,
  border: 'none',
  background: '#90caf9',
  color: '#000',
  cursor: 'pointer',
}

export const ShuffleAllButton = ({ filters, variant, className }) => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const notify = useNotify()
  const [loading, setLoading] = useState(false)
  const listLabel = translate('resources.song.actions.shuffleAll')
  const iconLabel = translate('menu.playRandom', { _: listLabel })
  const queryFilters = { ...filters, missing: false }

  const handleOnClick = () => {
    if (loading) {
      return
    }
    setLoading(true)
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter: queryFilters,
      })
      .then((res) => {
        const data = {}
        res.data.forEach((song) => {
          data[song.id] = song
        })
        dispatch(shuffleTracks(data))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
      .finally(() => {
        setLoading(false)
      })
  }

  if (variant === 'hero') {
    return (
      <button
        type="button"
        onClick={handleOnClick}
        className={className}
        style={heroStyle}
        disabled={loading}
        data-testid="shuffle-all-hero"
      >
        {iconLabel}
      </button>
    )
  }

  if (variant === 'icon') {
    return (
      <Tooltip title={iconLabel}>
        <IconButton
          color="inherit"
          onClick={handleOnClick}
          aria-label={iconLabel}
          disabled={loading}
          data-testid="title-shuffle-button"
        >
          <ShuffleIcon />
        </IconButton>
      </Tooltip>
    )
  }

  return (
    <Button
      onClick={handleOnClick}
      label={listLabel}
      disabled={loading}
      data-testid="shuffle-all-button"
    >
      <ShuffleIcon />
    </Button>
  )
}

ShuffleAllButton.propTypes = {
  filters: PropTypes.object,
  variant: PropTypes.oneOf(['default', 'icon', 'hero']),
  className: PropTypes.string,
}
ShuffleAllButton.defaultProps = {
  filters: {},
  variant: 'default',
}
