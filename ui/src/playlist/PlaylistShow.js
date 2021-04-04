import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Pagination as RaPagination,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import PlaylistDetails from './PlaylistDetails'
import PlaylistSongs from './PlaylistSongs'
import PlaylistActions from './PlaylistActions'
import { Title, isReadOnly } from '../common'
const useStyles = makeStyles(
  (theme) => ({
    playlistActions: {},
  }),
  {
    name: 'NDPlaylistShow',
  }
)

const PlaylistShowLayout = (props) => {
  const { loading, ...context } = useShowContext(props)
  const { record } = context
  const classes = useStyles()

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
          perPage={100}
          filter={{ playlist_id: props.id }}
        >
          <PlaylistSongs
            {...props}
            readOnly={isReadOnly(record.owner)}
            title={<Title subTitle={record.name} />}
            actions={
              <PlaylistActions
                className={classes.playlistActions}
                record={record}
              />
            }
            resource={'playlistTrack'}
            exporter={false}
            pagination={<RaPagination rowsPerPageOptions={[100, 250, 500]} />}
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
