import React from 'react'
import {
  Filter,
  FunctionField,
  NumberField,
  SearchInput,
  TextField,
  useListContext,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import {
  DurationField,
  List,
  SongContextMenu,
  SongDatagrid,
  SongDetails,
  SongTitleField,
  RatingField,
} from '../common'
import { useDispatch } from 'react-redux'
import { playTracks } from '../actions'
import { SongBulkActions } from '../common'
import { AlbumLinkField } from '../song/AlbumLinkField'
import { AddToPlaylistDialog } from '../dialogs'
import { makeStyles } from '@material-ui/core/styles'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import config from '../config'
import { QualityInfo } from '../common/QualityInfo'
import { FavouriteSongActions } from './FavouriteSongActions'
const useStyles = makeStyles((theme) => ({
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
    visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
  },
  ratingField: {
    visibility: 'hidden',
  },
}))

const SongFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="title" alwaysOn />
  </Filter>
)

const FavouriteSongs = ({ isXsmall }) => {
  const { data, ids } = useListContext()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const classes = useStyles({ isDesktop })
  const dispatch = useDispatch()

  return (
    <SongDatagrid
      expand={!isXsmall && <SongDetails />}
      rowClick={(id) => dispatch(playTracks(data, ids, id))}
      contextAlwaysVisible={!isDesktop}
      classes={{ row: classes.row }}
      hasBulkActions={true}
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
      {isDesktop && <TextField source="artist" />}
      {isDesktop && <NumberField source="trackNumber" />}
      {isDesktop && <NumberField source="playCount" sortByOrder={'DESC'} />}
      {isDesktop && (
        <FunctionField
          source="year"
          render={(r) => r.year || ''}
          sortByOrder={'DESC'}
        />
      )}
      {isDesktop && <QualityInfo source="quality" sortable={false} />}
      <DurationField source="duration" />
      {isDesktop && config.enableStarRating && (
        <RatingField
          source="rating"
          sortByOrder={'DESC'}
          resource={'song'}
          className={classes.ratingField}
        />
      )}
      <SongContextMenu
        className={classes.contextMenu}
        resource="favoriteSongs"
        label={
          <FavoriteBorderIcon
            fontSize={'small'}
            className={classes.contextHeader}
          />
        }
        refreshPage={true}
      />
    </SongDatagrid>
  )
}

const FavouriteSongList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
    <>
      <List
        {...props}
        sort={{ field: 'title', order: 'ASC' }}
        exporter={false}
        bulkActionButtons={<SongBulkActions />}
        actions={<FavouriteSongActions />}
        filters={<SongFilter />}
        filter={{ starred: true }}
        perPage={isXsmall ? 50 : 15}
      >
        <FavouriteSongs isXsmall={isXsmall} />
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default FavouriteSongList
