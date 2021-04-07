import React from 'react'
import Paper from '@material-ui/core/Paper'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import inflection from 'inflection'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import TableRow from '@material-ui/core/TableRow'
import {
  BooleanField,
  Datagrid,
  DateField,
  NumberField,
  Show,
  TextField,
  useTranslate,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { makeStyles } from '@material-ui/core/styles'
import {
  ArtistLinkField,
  DurationField,
  RangeField,
  SimpleList,
  MultiLineTextField,
  AlbumContextMenu,
  RatingField,
} from '../common'
import config from '../config'

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
  headerCell: {
    position: 'static',
  },
  ratingField: {
    visibility: 'hidden',
  },
})

const AlbumDetails = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const { record } = props
  const data = {
    albumArtist: <TextField record={record} source="albumArtist" />,
    genre: <TextField record={record} source="genre" />,
    compilation: <BooleanField record={record} source="compilation" />,
    updatedAt: <DateField record={record} source="updatedAt" showTime />,
    comment: <MultiLineTextField record={record} source="comment" />,
  }
  if (!record.comment) {
    delete data.comment
  }
  return (
    <Show {...props} title=" ">
      <TableContainer component={Paper}>
        <Table aria-label="album details" size="small">
          <TableBody>
            {Object.keys(data).map((key) => {
              return (
                <TableRow key={`${record.id}-${key}`}>
                  <TableCell
                    component="th"
                    scope="row"
                    className={classes.tableCell}
                  >
                    {translate(`resources.album.fields.${key}`, {
                      _: inflection.humanize(inflection.underscore(key)),
                    })}
                    :
                  </TableCell>
                  <TableCell align="left">{data[key]}</TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </Show>
  )
}

const AlbumListView = ({
  hasShow,
  hasEdit,
  hasList,
  syncWithLocation,
  ...rest
}) => {
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
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
          <RangeField record={r} source={'year'} sortBy={'maxYear'} />
          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
        </>
      )}
      linkType={'show'}
      rightIcon={(r) => <AlbumContextMenu record={r} />}
      {...rest}
    />
  ) : (
    <Datagrid
      expand={<AlbumDetails />}
      rowClick={'show'}
      classes={{ row: classes.row, headerCell: classes.headerCell }}
      {...rest}
    >
      <TextField source="name" />
      <ArtistLinkField source="artist" />
      {isDesktop && <NumberField source="songCount" sortByOrder={'DESC'} />}
      {isDesktop && <NumberField source="playCount" sortByOrder={'DESC'} />}
      <RangeField source={'year'} sortBy={'maxYear'} sortByOrder={'DESC'} />
      {isDesktop && <DurationField source="duration" />}
      {config.enableStarRating && (
        <RatingField
          source={'rating'}
          resource={'album'}
          sortByOrder={'DESC'}
          className={classes.ratingField}
        />
      )}
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

export default AlbumListView
