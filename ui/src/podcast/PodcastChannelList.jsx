import React from 'react'
import {
  CreateButton,
  Datagrid,
  DateField,
  EditButton,
  Filter,
  sanitizeListRestProps,
  SearchInput,
  TextField,
  TopToolbar,
  useRefresh,
  useTranslate,
} from 'react-admin'
import { Avatar, Box, makeStyles, Typography } from '@material-ui/core'
import MicIcon from '@material-ui/icons/Mic'
import { List, Title } from '../common'
import subsonic from '../subsonic'
import config from '../config'
import TopFeedsSuggestions from './TopFeedsSuggestions'

const useStyles = makeStyles({
  avatar: {
    width: 32,
    height: 32,
  },
  emptyRoot: {
    textAlign: 'center',
    padding: '3rem 1rem',
  },
  suggestions: {
    display: 'flex',
    justifyContent: 'center',
    marginTop: '1.5rem',
  },
})

const PodcastChannelFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="title" alwaysOn />
  </Filter>
)

const PodcastChannelListActions = ({
  className,
  filters,
  resource,
  showFilter,
  displayedFilters,
  filterValues,
  isAdmin,
  ...rest
}) => {
  const translate = useTranslate()
  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {isAdmin && (
        <CreateButton basePath="/podcastChannel">
          {translate('ra.action.create')}
        </CreateButton>
      )}
    </TopToolbar>
  )
}

const CoverArtField = ({ record }) => {
  const classes = useStyles()
  if (!record) return null
  const src =
    record.uploadedImage || record.coverArtUrl
      ? subsonic.getCoverArtUrl(record, config.uiCoverArtSize, true)
      : undefined
  return (
    <Avatar src={src} variant="rounded" className={classes.avatar}>
      <MicIcon fontSize="small" />
    </Avatar>
  )
}
CoverArtField.defaultProps = { label: '' }

const EmptyPodcastList = ({ basePath }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const refresh = useRefresh()

  return (
    <Box className={classes.emptyRoot}>
      <MicIcon fontSize="large" color="disabled" />
      <Typography variant="h6" gutterBottom>
        {translate('resources.podcastChannel.emptyTitle')}
      </Typography>
      <Typography variant="body2" color="textSecondary" gutterBottom>
        {translate('resources.podcastChannel.emptyBody')}
      </Typography>
      <Box mt={2}>
        <CreateButton basePath={basePath || '/podcastChannel'}>
          {translate('resources.podcastChannel.search')}
        </CreateButton>
      </Box>
      <Box className={classes.suggestions}>
        <TopFeedsSuggestions onSubscribed={refresh} />
      </Box>
    </Box>
  )
}

const PodcastChannelList = ({ permissions, ...props }) => {
  const translate = useTranslate()
  const isAdmin = permissions === 'admin'

  return (
    <List
      {...props}
      exporter={false}
      title={<Title title={translate('menu.podcasts')} />}
      sort={{ field: 'title', order: 'ASC' }}
      bulkActionButtons={isAdmin ? undefined : false}
      hasCreate={isAdmin}
      actions={<PodcastChannelListActions isAdmin={isAdmin} />}
      filters={<PodcastChannelFilter />}
      empty={isAdmin ? <EmptyPodcastList /> : false}
      perPage={25}
    >
      <Datagrid rowClick="show">
        <CoverArtField source="id" sortable={false} />
        <TextField source="title" />
        <TextField source="status" />
        <TextField source="downloadPolicy" />
        <DateField source="lastCheckedAt" showTime />
        {isAdmin && <EditButton />}
      </Datagrid>
    </List>
  )
}

export default PodcastChannelList
