import React, { useEffect } from 'react'
import { useHistory } from 'react-router-dom'
import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
  ImageField,
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
} from '../common'
import { fetchArtistInfoExtra } from '../subsonic'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'

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
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
  artistImage: {
    width: '100px',
    [theme.breakpoints.up('lg')]: {
      width: '150px',
    },
    height: 'auto',
    '& img': {
      width: '100%',
      height: '100%',
      borderRadius: '50%',
      backgroundColor: 'red',
    },
  },
}))

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
  useEffect(() => {
    const artists = rest.data
    if (artists !== []) {
      for (const id in artists) {
        let artist = artists[id]
        if (artist.smallImageUrl === '') {
          fetchArtistInfoExtra(id)
        }
      }
    }
  }, [rest.data])

  const classes = useStyles()
  const handleArtistLink = useGetHandleArtistClick(width)
  const history = useHistory()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return isXsmall ? (
    <ArtistSimpleList
      linkType={(id) => history.push(handleArtistLink(id))}
      {...rest}
    />
  ) : (
    <Datagrid rowClick={handleArtistLink} classes={{ row: classes.row }}>
      <ImageField
        source="smallImageUrl"
        label="Image"
        className={classes.artistImage}
      />
      <TextField source="name" />
      <NumberField source="albumCount" sortByOrder={'DESC'} />
      <NumberField source="songCount" sortByOrder={'DESC'} />
      <NumberField source="playCount" sortByOrder={'DESC'} />
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
      >
        <ArtistListView {...props} />
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(ArtistList)
