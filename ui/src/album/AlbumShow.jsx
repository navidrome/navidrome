import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Title as RaTitle,
} from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import AlbumSongs from './AlbumSongs'
import AlbumDetails from './AlbumDetails'
import AlbumActions from './AlbumActions'
import { useResourceRefresh, Title } from '../common'

const useStyles = makeStyles(
  (theme) => ({
    albumActions: {
      width: '100%',
    },
  }),
  {
    name: 'NDAlbumShow',
  },
)

const AlbumShowLayout = (props) => {
  const { loading, ...context } = useShowContext(props)
  const { record } = context
  const classes = useStyles()
  useResourceRefresh('album', 'song')

  return (
    <>
      {record && <RaTitle title={<Title subTitle={record.name} />} />}
      {record && <AlbumDetails {...context} />}
      {record && (
        <ReferenceManyField
          {...context}
          addLabel={false}
          reference="song"
          target="album_id"
          sort={{ field: 'album', order: 'ASC' }}
          perPage={0}
          pagination={null}
        >
          <AlbumSongs
            resource={'song'}
            exporter={false}
            album={record}
            actions={
              <AlbumActions className={classes.albumActions} record={record} />
            }
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
