import React, { useEffect } from 'react'
import { useHistory } from 'react-router-dom'
import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { useMediaQuery, withWidth } from '@material-ui/core'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { AddToPlaylistDialog } from '../dialogs'
import {
  ArtistContextMenu,
  List,
  QuickFilter,
  useGetHandleArtistClick,
  ArtistSimpleList,
  RatingField,
} from '../common'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import ArtistListActions from './ArtistListActions'
import useSelectedFields from '../common/useSelectedFields'
import { setOmittedFields } from '../actions'
import { useDispatch } from 'react-redux'

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

const ArtistFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput source="name" alwaysOn />
    {config.enableFavourites && (
      <QuickFilter
        source="starred"
        label={<FavoriteIcon fontSize={'small'} />}
        defaultValue={true}
      />
    )}
  </Filter>
)

const ArtistListView = ({ hasShow, hasEdit, hasList, width, ...rest }) => {
  const classes = useStyles()
  const handleArtistLink = useGetHandleArtistClick(width)
  const history = useHistory()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const dispatch = useDispatch()

  const toggleableFields = {
    albumCount: <NumberField source="albumCount" sortByOrder={'DESC'} />,
    songCount: <NumberField source="songCount" sortByOrder={'DESC'} />,
    playCount: <NumberField source="playCount" sortByOrder={'DESC'} />,
    rating: config.enableStarRating && (
      <RatingField
        source="rating"
        sortByOrder={'DESC'}
        resource={'artist'}
        className={classes.ratingField}
      />
    ),
  }

  const [columns, omitted] = useSelectedFields({
    resource: 'artist',
    columns: toggleableFields,
  })

  useEffect(() => {
    dispatch(setOmittedFields(omitted))
  }, [omitted])
  return isXsmall ? (
    <ArtistSimpleList
      linkType={(id) => history.push(handleArtistLink(id))}
      {...rest}
    />
  ) : (
    <Datagrid rowClick={handleArtistLink} classes={{ row: classes.row }}>
      <TextField source="name" />
      {columns}
      <ArtistContextMenu
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
    </Datagrid>
  )
}

const ArtistList = (props) => {
  return (
    <>
      <List
        {...props}
        sort={{ field: 'name', order: 'ASC' }}
        exporter={false}
        bulkActionButtons={false}
        filters={<ArtistFilter />}
        actions={<ArtistListActions />}
      >
        <ArtistListView {...props} />
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(ArtistList)
