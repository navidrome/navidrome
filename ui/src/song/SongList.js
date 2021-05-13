import React from 'react'
import {
  Filter,
  FunctionField,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import FavoriteIcon from '@material-ui/icons/Favorite'
import {
  DurationField,
  List,
  SongContextMenu,
  SongDatagrid,
  SongDetails,
  QuickFilter,
  SongTitleField,
  SongSimpleList,
  RatingField,
} from '../common'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { SongBulkActions } from '../common'
import { SongListActions } from './SongListActions'
import { AlbumLinkField } from './AlbumLinkField'
import { AddToPlaylistDialog } from '../dialogs'
import { makeStyles } from '@material-ui/core/styles'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import config from '../config'
import { QualityInfo } from '../common/QualityInfo'

const useStyles = makeStyles({
  contextHeader: {
    marginLeft: '3px',
    marginTop: '-2px',
    verticalAlign: 'text-top',
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
    visibility: 'hidden',
  },
  ratingField: {
    visibility: 'hidden',
  },
})

const SongFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="title" alwaysOn />
    {config.enableFavourites && (
      <QuickFilter
        source="starred"
        label={<FavoriteIcon fontSize={'small'} />}
        defaultValue={true}
      />
    )}
  </Filter>
)

const SongList = (props) => {
  const classes = useStyles()
  const dispatch = useDispatch()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  const handleRowClick = (id, basePath, record) => {
    dispatch(setTrack(record))
  }

  return (
    <>
      <List
        {...props}
        sort={{ field: 'title', order: 'ASC' }}
        exporter={false}
        bulkActionButtons={<SongBulkActions />}
        actions={<SongListActions />}
        filters={<SongFilter />}
        perPage={isXsmall ? 50 : 15}
      >
        {isXsmall ? (
          <SongSimpleList />
        ) : (
          <SongDatagrid
            expand={<SongDetails />}
            rowClick={handleRowClick}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            <SongTitleField source="title" showTrackNumbers={false} />
            {isDesktop && (
              <AlbumLinkField
                source="album"
                sortBy={
                  'album, order_album_artist_name, disc_number, track_number, title'
                }
                sortByOrder={'ASC'}
              />
            )}
            <TextField source="artist" />
            {isDesktop && <NumberField source="trackNumber" />}
            {isDesktop && (
              <NumberField source="playCount" sortByOrder={'DESC'} />
            )}
            {isDesktop && (
              <FunctionField
                source="year"
                render={(r) => r.year || ''}
                sortByOrder={'DESC'}
              />
            )}
            {isDesktop && <QualityInfo source="quality" sortable={false} />}
            <DurationField source="duration" />
            {isDesktop && <NumberField source="bpm" />}
            {config.enableStarRating && (
              <RatingField
                source="rating"
                sortByOrder={'DESC'}
                resource={'song'}
                className={classes.ratingField}
              />
            )}
            <SongContextMenu
              source={'starred'}
              sortBy={'starred ASC, starredAt ASC'}
              sortByOrder={'DESC'}
              sortable={config.enableFavourites}
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
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default SongList
