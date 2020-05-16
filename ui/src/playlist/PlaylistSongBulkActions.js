import React, { Fragment, useEffect } from 'react'
import { BulkDeleteButton, useUnselectAll } from 'react-admin'
import PropTypes from 'prop-types'

const PlaylistSongBulkActions = ({ playlistId, ...rest }) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('playlistTrack')
    // eslint-disable-next-line
  }, [])
  return (
    <Fragment>
      <BulkDeleteButton {...rest} resource={`playlist/${playlistId}/tracks`} />
    </Fragment>
  )
}

PlaylistSongBulkActions.propTypes = {
  playlistId: PropTypes.string.isRequired,
}

export default PlaylistSongBulkActions
