import React, { useMemo } from 'react'
import {
  BulkActionsToolbar,
  ListToolbar,
  TextField,
  NumberField,
  useVersion,
  useListContext,
  FunctionField,
} from 'react-admin'
import clsx from 'clsx'
import { useDispatch } from 'react-redux'
import { Card, useMediaQuery } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { playTracks } from '../actions'
import {
  DurationField,
  SongBulkActions,
  SongContextMenu,
  SongDatagrid,
  SongInfo,
  SongTitleField,
  RatingField,
  QualityInfo,
  useSelectedFields,
  useResourceRefresh,
  DateField,
  SizeField,
  ArtistLinkField,
} from '../common'
import config from '../config'
import ExpandInfoDialog from '../dialogs/ExpandInfoDialog'

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
    actions: {
      zIndex: 2,
      display: 'flex',
      justifyContent: 'flex-end',
      flexWrap: 'wrap',
    },
    noResults: { padding: 20 },
    columnIcon: {
      marginLeft: '3px',
      marginTop: '-2px',
      verticalAlign: 'text-top',
    },
    toolbar: {
      justifyContent: 'flex-start',
    },
    row: {
      '&:hover': {
        '& $contextMenu': {
          visibility: 'visible',
        },
        '& $ratingField': {
          visibility: 'visible',
        },
      },
    },
    contextMenu: {
      visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
    },
    ratingField: {
      visibility: 'hidden',
    },
  }),
  { name: 'RaList' },
)

const AlbumSongs = (props) => {
  const { data, ids } = props
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()
  const version = useVersion()
  useResourceRefresh('song', 'album')

  const toggleableFields = useMemo(() => {
    return {
      trackNumber: isDesktop && (
        <TextField
          source="trackNumber"
          sortBy="releaseDate asc, discNumber asc, trackNumber asc"
          label="#"
          sortable={false}
        />
      ),
      title: (
        <SongTitleField
          source="title"
          sortable={false}
          showTrackNumbers={!isDesktop}
        />
      ),
      artist: isDesktop && <ArtistLinkField source="artist" />,
      duration: <DurationField source="duration" sortable={false} />,
      year: isDesktop && (
        <FunctionField
          source="year"
          render={(r) => r.year || ''}
          sortByOrder={'DESC'}
        />
      ),
      playCount: isDesktop && (
        <NumberField source="playCount" sortable={false} />
      ),
      playDate: <DateField source="playDate" sortable={false} showTime />,
      quality: isDesktop && <QualityInfo source="quality" sortable={false} />,
      size: isDesktop && <SizeField source="size" sortable={false} />,
      channels: isDesktop && <NumberField source="channels" sortable={false} />,
      bpm: isDesktop && <NumberField source="bpm" sortable={false} />,
      rating: isDesktop && config.enableStarRating && (
        <RatingField
          resource={'song'}
          source="rating"
          sortable={false}
          className={classes.ratingField}
        />
      ),
    }
  }, [isDesktop, classes.ratingField])

  const columns = useSelectedFields({
    resource: 'albumSong',
    columns: toggleableFields,
    omittedColumns: ['title'],
    defaultOff: ['channels', 'bpm', 'year', 'playCount', 'playDate', 'size'],
  })

  const bulkActionsLabel = isDesktop
    ? 'ra.action.bulk_actions'
    : 'ra.action.bulk_actions_mobile'

  return (
    <>
      <ListToolbar
        classes={{ toolbar: classes.toolbar }}
        actions={props.actions}
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
            showReleaseDivider={true}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            {columns}
            <SongContextMenu
              source={'starred'}
              sortable={false}
              className={classes.contextMenu}
              label={
                config.enableFavourites && (
                  <FavoriteBorderIcon
                    fontSize={'small'}
                    className={classes.columnIcon}
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

export const removeAlbumCommentsFromSongs = ({ album, data }) => {
  if (album?.comment && data) {
    Object.values(data).forEach((song) => {
      song.comment = ''
    })
  }
}

const SanitizedAlbumSongs = (props) => {
  removeAlbumCommentsFromSongs(props)
  const { loaded, loading, total, ...rest } = useListContext(props)
  return <>{loaded && <AlbumSongs {...rest} actions={props.actions} />}</>
}

export default SanitizedAlbumSongs
