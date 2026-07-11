import React from 'react'
import {
  ReferenceManyField,
  ShowContextProvider,
  useShowContext,
  useShowController,
  Title as RaTitle,
  Datagrid,
  TextField,
  DateField,
  FunctionField,
  SimpleShowLayout,
  useTranslate,
  Pagination,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { Typography, Box, Avatar, makeStyles } from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import { Title } from '../common'
import { setTrack } from '../actions'
import subsonic from '../subsonic'
import config from '../config'
import { songFromPodcastEpisode } from './helper'

const useStyles = makeStyles({
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '1rem',
    marginBottom: '1rem',
  },
  cover: {
    width: '5rem',
    height: '5rem',
  },
})

const PodcastChannelHeader = () => {
  const { record } = useShowContext()
  const classes = useStyles()
  if (!record) return null
  const cover =
    record.uploadedImage || record.coverArtUrl
      ? subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)
      : undefined
  return (
    <Box className={classes.header}>
      <Avatar src={cover} variant="rounded" className={classes.cover}>
        <MicIcon />
      </Avatar>
      <Box>
        <Typography variant="h6">{record.title}</Typography>
        {record.description && (
          <Typography variant="body2" color="textSecondary">
            {record.description}
          </Typography>
        )}
      </Box>
    </Box>
  )
}

const EpisodesSection = ({ channel }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleRowClick = (id, basePath, record) => {
    dispatch(setTrack(songFromPodcastEpisode(record, channel)))
    return false
  }

  return (
    <>
      <Box mt={2} mb={1}>
        <Typography variant="h6">
          {translate('resources.podcastEpisode.name', { smart_count: 2 })}
        </Typography>
      </Box>
      <Datagrid rowClick={handleRowClick}>
        <TextField source="title" />
        <DateField source="publishDate" />
        <FunctionField
          source="duration"
          render={(record) => {
            if (!record.duration) return null
            const mins = Math.floor(record.duration / 60)
            const secs = Math.floor(record.duration % 60)
              .toString()
              .padStart(2, '0')
            return `${mins}:${secs}`
          }}
        />
        <FunctionField
          source="downloadStatus"
          render={(record) =>
            translate(
              `resources.podcastEpisode.downloadStatus.${record.downloadStatus}`,
              { _: record.downloadStatus },
            )
          }
        />
      </Datagrid>
    </>
  )
}

const PodcastChannelShowLayout = (props) => {
  const { record, loading } = useShowContext(props)

  if (loading || !record) return null

  return (
    <>
      <RaTitle title={<Title subTitle={record.title} />} />
      <SimpleShowLayout>
        <PodcastChannelHeader />
        <ReferenceManyField
          reference="podcastEpisode"
          target="channel_id"
          label=""
          sort={{ field: 'publishDate', order: 'DESC' }}
          perPage={100}
          pagination={<Pagination rowsPerPageOptions={[50, 100, 250]} />}
          fullWidth
        >
          <EpisodesSection channel={record} />
        </ReferenceManyField>
      </SimpleShowLayout>
    </>
  )
}

const PodcastChannelShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <PodcastChannelShowLayout {...props} {...controllerProps} />
    </ShowContextProvider>
  )
}

export default PodcastChannelShow
