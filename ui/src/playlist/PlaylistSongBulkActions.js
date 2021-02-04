import React, { Fragment, useEffect } from 'react'
import {
  BulkDeleteButton,
  useUnselectAll,
  ResourceContextProvider,
} from 'react-admin'
import PropTypes from 'prop-types'

// Replace original resource with "fake" one for removing tracks from playlist
const PlaylistSongBulkActions = ({ playlistId, resource, ...rest }) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('playlistTrack')
    // eslint-disable-next-line
  }, [])

  const mappedResource = `playlist/${playlistId}/tracks`
  return (
    <ResourceContextProvider value={mappedResource}>
      <Fragment>
        <BulkDeleteButton {...rest} resource={mappedResource} />
      </Fragment>
    </ResourceContextProvider>
  )
}

PlaylistSongBulkActions.propTypes = {
  playlistId: PropTypes.string.isRequired,
}

export default PlaylistSongBulkActions
