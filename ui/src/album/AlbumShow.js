import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
} from 'react-admin'
import AlbumSongs from './AlbumSongs'
import AlbumDetails from './AlbumDetails'
import AlbumActions from './AlbumActions'

const AlbumShowLayout = (props) => {
  const { loading, ...context } = useShowContext(props)
  const { record } = context

  return (
    <>
      {record && <AlbumDetails {...context} />}
      {record && (
        <ReferenceManyField
          {...context}
          addLabel={false}
          reference="albumSong"
          target="album_id"
          sort={{ field: 'discNumber asc, trackNumber asc', order: 'ASC' }}
          perPage={0}
          pagination={null}
        >
          <AlbumSongs
            resource={'albumSong'}
            exporter={false}
            actions={<AlbumActions record={record} />}
          />
        </ReferenceManyField>
      )}
    </>
  )
}

const AlbumShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <AlbumShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

export default AlbumShow
