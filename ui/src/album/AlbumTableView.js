import React, { useMemo } from 'react'
import { NumberField, TextField } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { makeStyles } from '@material-ui/core/styles'
import {
  ArtistLinkField,
  DurationField,
  RangeField,
  SimpleList,
  AlbumContextMenu,
  RatingField,
  useSelectedFields,
} from '../common'
import config from '../config'
import { Datagrid } from '../infiniteScroll'

const useStyles = makeStyles({
  columnIcon: {
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
  tableCell: {
    width: '17.5%',
  },
  contextMenu: {
    visibility: 'hidden',
  },
  ratingField: {
    visibility: 'hidden',
  },
})

const AlbumTableView = ({
  hasShow,
  hasEdit,
  hasList,
  syncWithLocation,
  ...rest
}) => {
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  const toggleableFields = useMemo(() => {
    return {
      artist: <ArtistLinkField source="artist" />,
      songCount: isDesktop && (
        <NumberField source="songCount" sortByOrder={'DESC'} />
      ),
      playCount: isDesktop && (
        <NumberField source="playCount" sortByOrder={'DESC'} />
      ),
      year: (
        <RangeField
          source={'year'}
          sortBy={'maxYear'}
          sortByOrder={'DESC'}
          dataKey={'maxYear'}
        />
      ),
      duration: isDesktop && <DurationField source="duration" />,
      rating: config.enableStarRating && (
        <RatingField
          source={'rating'}
          resource={'album'}
          sortByOrder={'DESC'}
          className={classes.ratingField}
        />
      ),
    }
  }, [classes.ratingField, isDesktop])

  const columns = useSelectedFields({
    resource: 'album',
    columns: toggleableFields,
  })

  return isXsmall ? (
    <SimpleList
      primaryText={(r) => r.name}
      secondaryText={(r) => (
        <>
          {r.albumArtist}
          {config.enableStarRating && (
            <>
              <br />
              <RatingField
                record={r}
                sortByOrder={'DESC'}
                source={'rating'}
                resource={'album'}
                size={'small'}
              />
            </>
          )}
        </>
      )}
      tertiaryText={(r) => (
        <>
          <RangeField
            record={r}
            source={'year'}
            sortBy={'maxYear'}
            dataKey={'maxYear'}
          />
          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
        </>
      )}
      linkType={'show'}
      rightIcon={(r) => <AlbumContextMenu record={r} />}
      {...rest}
    />
  ) : (
    <Datagrid rowClick={'show'} classes={{ row: classes.row }} {...rest}>
      <TextField source="name" flexgrow={0.75} width={200} />
      {columns}
      <AlbumContextMenu
        source={'starred'}
        sortBy={'starred ASC, starredAt ASC'}
        sortByOrder={'DESC'}
        sortable={config.enableFavourites}
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
    </Datagrid>
  )
}

export default AlbumTableView
