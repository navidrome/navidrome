import React from 'react'
import {
  BulkActionsToolbar,
  useListContext,
  useVersion,
  Filter,
  SearchInput,
  ListToolbar,
} from 'react-admin'
import clsx from 'clsx'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { playTracks } from '../actions'
import {
  SongBulkActions,
  SongContextMenu,
  SongDatagrid,
  SongInfo,
  SongTitleField,
  ArtistLinkField,
  DurationField,
  useResourceRefresh,
  useSelectedFields,
} from '../common'
import config from '../config'
import ExpandInfoDialog from '../dialogs/ExpandInfoDialog'
import FolderActions from './FolderActions'

const useStyles = makeStyles(
  (theme) => ({
    root: {},
    main: {
      display: 'flex',
    },
    content: {
      marginTop: 0,
      transition: theme.transitions.create('margin-top'),
      position: 'relative',
      flex: '1 1 auto',
      [theme.breakpoints.down('xs')]: {
        boxShadow: 'none',
      },
    },
    bulkActionsDisplayed: {
      marginTop: -theme.spacing(8),
      transition: theme.transitions.create('margin-top'),
    },
    row: {
      '&:hover': {
        '& $contextMenu': {
          visibility: 'visible',
        },
      },
    },
    contextMenu: {
      visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
    },
    contextHeader: {
      marginLeft: '3px',
      marginTop: '-2px',
      verticalAlign: 'text-top',
    },
    toolbar: {
      justifyContent: 'flex-start',
    },
  }),
  { name: 'FolderSongs' },
)

const SongFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="title" alwaysOn />
  </Filter>
)

const FolderSongsContent = (props) => {
  const { data, ids, folder } = props
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()
  const version = useVersion()
  useResourceRefresh('song', 'folder')

  const toggleableFields = React.useMemo(() => {
    return {
      title: (
        <SongTitleField
          source="title"
          sortable={false}
          showTrackNumbers={false}
        />
      ),
      artist: isDesktop && <ArtistLinkField source="artist" sortable={false} />,
      duration: <DurationField source="duration" sortable={false} />,
    }
  }, [isDesktop])

  const columns = useSelectedFields({
    resource: 'folderSong',
    columns: toggleableFields,
    omittedColumns: ['title'],
    defaultOff: [],
  })

  const bulkActionsLabel = isDesktop
    ? 'ra.action.bulk_actions'
    : 'ra.action.bulk_actions_mobile'

  return (
    <>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        filters={<SongFilter />}
        actions={<FolderActions record={folder} />}
        {...props}
      />
      <div className={classes.main}>
        <Card
          className={clsx(classes.content, {
            [classes.bulkActionsDisplayed]: props.selectedIds.length > 0,
          })}
          key={version}
        >
          <BulkActionsToolbar {...props} label={bulkActionsLabel}>
            <SongBulkActions />
          </BulkActionsToolbar>
          <SongDatagrid
            rowClick={(id) => dispatch(playTracks(data, ids, id))}
            {...props}
            hasBulkActions={true}
            showDiscSubtitles={true}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            {columns}
            <SongContextMenu
              source={'starred_at'}
              sortable={false}
              className={classes.contextMenu}
              label={
                config.enableFavourites && (
                  <FavoriteBorderIcon
                    fontSize={'small'}
                    className={classes.contextHeader}
                  />
                )
              }
            />
          </SongDatagrid>
        </Card>
      </div>
      <ExpandInfoDialog content={<SongInfo />} />
    </>
  )
}

const FolderSongs = (props) => {
  const { loaded, loading, total, ...rest } = useListContext(props)
  return <>{loaded && <FolderSongsContent {...rest} folder={props.folder} />}</>
}

export default FolderSongs
