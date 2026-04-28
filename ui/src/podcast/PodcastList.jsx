import React from 'react'
import { Avatar, ButtonGroup, IconButton, makeStyles, Tooltip, Typography, useMediaQuery } from '@material-ui/core'
import FileCopyIcon from '@material-ui/icons/FileCopy'
import MicIcon from '@material-ui/icons/Mic'
import RefreshIcon from '@material-ui/icons/Refresh'
import ViewModuleIcon from '@material-ui/icons/ViewModule'
import ViewHeadlineIcon from '@material-ui/icons/ViewHeadline'
import {
  Button,
  CreateButton,
  Datagrid,
  DateField,
  Filter,
  sanitizeListRestProps,
  SearchInput,
  SimpleList,
  TextField,
  TopToolbar,
  useNotify,
  useRefresh,
  useTranslate,
} from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { List } from '../common'
import subsonic from '../subsonic'
import StatusBadge from './StatusBadge'
import PodcastGridView from './PodcastGridView'
import { podcastViewGrid, podcastViewTable } from '../actions'

const useStyles = makeStyles({
  row: { '&:hover': { '& $contextMenu': { visibility: 'visible' } } },
  contextMenu: { visibility: 'hidden' },
  toggleTitle: { margin: '1rem' },
  buttonGroup: { width: '100%', justifyContent: 'center' },
  leftButton: { paddingRight: '0.5rem' },
  rightButton: { paddingLeft: '0.5rem' },
})

const PodcastFilter = (props) => (
  <Filter {...props} variant="outlined">
    <SearchInput id="search" source="title" alwaysOn />
  </Filter>
)

const bestImageUrl = (record, targetWidth) => {
  if (!record.images || record.images.length === 0) return record.imageUrl
  const sorted = [...record.images].sort((a, b) => a.width - b.width)
  const best = sorted.find((img) => img.width >= targetWidth) || sorted[sorted.length - 1]
  return best ? best.url : record.imageUrl
}

const CoverArtField = ({ record }) => {
  if (!record) return null
  if (record.imageUrl) {
    return (
      <Avatar src={bestImageUrl(record, 55)} variant="rounded" style={{ width: 55, height: 55 }} alt={record.title} />
    )
  }
  return (
    <Avatar variant="rounded" style={{ width: 55, height: 55 }}>
      <MicIcon />
    </Avatar>
  )
}
CoverArtField.defaultProps = { label: '', sortable: false }

const FeedUrlField = ({ record }) => {
  const notify = useNotify()
  const translate = useTranslate()
  if (!record?.url) return null
  const handleCopy = (e) => {
    e.stopPropagation()
    navigator.clipboard.writeText(record.url)
    notify('resources.podcast.notifications.urlCopied')
  }
  return (
    <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      <span style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {record.url}
      </span>
      <Tooltip title={translate('resources.podcast.actions.copyUrl', { _: 'Copy URL' })}>
        <IconButton size="small" onClick={handleCopy}>
          <FileCopyIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </span>
  )
}
FeedUrlField.defaultProps = { label: 'resources.podcast.fields.url', sortable: false }

const StatusField = ({ record }) => {
  if (!record || record.status !== 'error') return null
  return <StatusBadge status="error" errorMessage={record.errorMessage} />
}
StatusField.defaultProps = { label: '' }

const RefreshButton = () => {
  const notify = useNotify()
  const refresh = useRefresh()

  const handleClick = async () => {
    await subsonic.refreshPodcasts()
    notify('resources.podcast.notifications.refreshStarted')
    refresh()
  }

  return (
    <Button onClick={handleClick} label="resources.podcast.actions.refresh">
      <RefreshIcon />
    </Button>
  )
}

const PodcastViewToggler = React.forwardRef(({ showTitle = true }, ref) => {
  const dispatch = useDispatch()
  const podcastView = useSelector((state) => state.podcastView)
  const classes = useStyles()
  const translate = useTranslate()
  return (
    <div ref={ref}>
      {showTitle && (
        <Typography className={classes.toggleTitle}>
          {translate('ra.toggleFieldsMenu.layout')}
        </Typography>
      )}
      <ButtonGroup variant="text" color="primary" className={classes.buttonGroup}>
        <Button
          size="small"
          className={classes.leftButton}
          label={translate('ra.toggleFieldsMenu.grid')}
          color={podcastView.grid ? 'primary' : 'secondary'}
          onClick={() => dispatch(podcastViewGrid())}
        >
          <ViewModuleIcon fontSize="inherit" />
        </Button>
        <Button
          size="small"
          className={classes.rightButton}
          label={translate('ra.toggleFieldsMenu.table')}
          color={podcastView.grid ? 'secondary' : 'primary'}
          onClick={() => dispatch(podcastViewTable())}
        >
          <ViewHeadlineIcon fontSize="inherit" />
        </Button>
      </ButtonGroup>
    </div>
  )
})
PodcastViewToggler.displayName = 'PodcastViewToggler'

const PodcastListActions = ({ className, filters, resource, showFilter, displayedFilters, filterValues, isAdmin, ...rest }) => {
  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {isAdmin && <RefreshButton />}
      {isAdmin && <CreateButton basePath="/podcast" />}
      {filters && React.cloneElement(filters, { resource, showFilter, displayedFilters, filterValues, context: 'button' })}
      <PodcastViewToggler showTitle={false} />
    </TopToolbar>
  )
}

const PodcastList = ({ permissions, ...props }) => {
  const classes = useStyles()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isAdmin = permissions === 'admin'
  const podcastView = useSelector((state) => state.podcastView)

  return (
    <List
      {...props}
      exporter={false}
      sort={{ field: 'title', order: 'ASC' }}
      bulkActionButtons={isAdmin ? undefined : false}
      hasCreate={isAdmin}
      actions={<PodcastListActions isAdmin={isAdmin} />}
      filters={<PodcastFilter />}
    >
      {isXsmall ? (
        <SimpleList
          leftAvatar={(r) => <CoverArtField record={r} />}
          primaryText={(r) => r.title}
          secondaryText={(r) => r.url}
        />
      ) : podcastView.grid ? (
        <PodcastGridView />
      ) : (
        <Datagrid rowClick="show" classes={{ row: classes.row }}>
          <CoverArtField source="id" />
          <TextField source="title" />
          <FeedUrlField source="url" />
          <StatusField source="status" sortable={false} />
          <DateField source="updatedAt" showTime />
        </Datagrid>
      )}
    </List>
  )
}

export default PodcastList
