import React, { Fragment, useEffect } from 'react'
import {
  BulkDeleteButton,
  useUnselectAll,
  ResourceContextProvider,
} from 'react-admin'
import PropTypes from 'prop-types'

// Replace original resource with "fake" one for removing tracks from playlist
const PlaylistSongBulkActions = ({
  playlistId,
  resource,
  onUnselectItems,
  ...rest
}) => {
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll('playlistTrack')
  }, [unselectAll])

  const mappedResource = `playlist/${playlistId}/tracks`
  return (
    <ResourceContextProvider value={mappedResource}>
      <Fragment>
        <BulkDeleteButton
          {...rest}
          resource={mappedResource}
          onClick={onUnselectItems}
        />
      </Fragment>
    </ResourceContextProvider>
  )
}

PlaylistSongBulkActions.propTypes = {
  playlistId: PropTypes.string.isRequired,
}

export default PlaylistSongBulkActions
