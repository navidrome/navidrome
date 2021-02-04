import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
} from 'react-admin'
import PlaylistDetails from './PlaylistDetails'
import PlaylistSongs from './PlaylistSongs'
import PlaylistActions from './PlaylistActions'
import { Title, isReadOnly } from '../common'

const PlaylistShowLayout = (props) => {
  const { loading, ...context } = useShowContext(props)
  const { record } = context
  return (
    <>
      {record && <PlaylistDetails {...context} />}
      {record && (
        <ReferenceManyField
          {...context}
          addLabel={false}
          reference="playlistTrack"
          target="playlist_id"
          sort={{ field: 'id', order: 'ASC' }}
          perPage={0}
          filter={{ playlist_id: props.id }}
        >
          <PlaylistSongs
            {...props}
            readOnly={isReadOnly(record.owner)}
            title={<Title subTitle={record.name} />}
            actions={<PlaylistActions record={record} />}
            resource={'playlistTrack'}
            exporter={false}
            perPage={0}
            pagination={null}
          />
        </ReferenceManyField>
      )}
    </>
  )
}

const PlaylistShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <PlaylistShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

export default PlaylistShow
