import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Pagination,
  Title as RaTitle,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import PlaylistDetails from './PlaylistDetails'
import PlaylistSongs from './PlaylistSongs'
import PlaylistActions from './PlaylistActions'
import { Title, canChangeTracks, useResourceRefresh } from '../common'

const useStyles = makeStyles(
  (theme) => ({
    playlistActions: {
      width: '100%',
    },
  }),
  {
    name: 'NDPlaylistShow',
  },
)

const PlaylistShowLayout = (props) => {
  const { loading, ...context } = useShowContext(props)
  const { record } = context
  const classes = useStyles()
  useResourceRefresh('song')

  return (
    <>
      {record && <RaTitle title={<Title subTitle={record.name} />} />}
      {record && <PlaylistDetails {...context} />}
      {record && (
        <ReferenceManyField
          {...context}
          addLabel={false}
          reference="playlistTrack"
          target="playlist_id"
          sort={{ field: 'id', order: 'ASC' }}
          perPage={100}
          filter={{ playlist_id: props.id }}
        >
          <PlaylistSongs
            {...props}
            readOnly={!canChangeTracks(record)}
            title={<Title subTitle={record.name} />}
            actions={
              <PlaylistActions
                className={classes.playlistActions}
                record={record}
              />
            }
            resource={'playlistTrack'}
            exporter={false}
            pagination={<Pagination rowsPerPageOptions={[100, 250, 500]} />}
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
