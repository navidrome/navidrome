import React from 'react'
import {
  Filter,
  FunctionField,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import StarIcon from '@material-ui/icons/Star'
import {
  DurationField,
  List,
  SimpleList,
  SongContextMenu,
  SongDatagrid,
  SongDetails,
  QuickFilter,
  SongTitleField,
} from '../common'
import { useDispatch } from 'react-redux'
import { setTrack } from '../actions'
import { SongBulkActions } from '../common'
import { SongListActions } from './SongListActions'
import { AlbumLinkField } from './AlbumLinkField'
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'
import { makeStyles } from '@material-ui/core/styles'
import StarBorderIcon from '@material-ui/icons/StarBorder'

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
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
})

const SongFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="title" alwaysOn />
    <QuickFilter
      source="starred"
      label={<StarIcon fontSize={'small'} />}
      defaultValue={true}
    />
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
          <SimpleList
            primaryText={(r) => r.title}
            secondaryText={(r) => r.artist}
            tertiaryText={(r) => (
              <>
                <DurationField record={r} source={'duration'} />
                &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
              </>
            )}
            linkType={(id, basePath, record) => dispatch(setTrack(record))}
            rightIcon={(r) => <SongContextMenu record={r} visible={true} />}
          />
        ) : (
          <SongDatagrid
            expand={<SongDetails />}
            rowClick={handleRowClick}
            contextAlwaysVisible={!isDesktop}
            classes={{ row: classes.row }}
          >
            <SongTitleField source="title" showTrackNumbers={false} />
            {isDesktop && <AlbumLinkField source="album" />}
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
            <DurationField source="duration" />
            <SongContextMenu
              source={'starred'}
              sortBy={'starred ASC, starredAt ASC'}
              sortByOrder={'DESC'}
              className={classes.contextMenu}
              label={
                <StarBorderIcon
                  fontSize={'small'}
                  className={classes.contextHeader}
                />
              }
              textAlign={'right'}
            />
          </SongDatagrid>
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default SongList
