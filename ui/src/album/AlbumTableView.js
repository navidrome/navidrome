import React, { useMemo } from 'react'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import inflection from 'inflection'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import TableRow from '@material-ui/core/TableRow'
import {
  ArrayField,
  BooleanField,
  ChipField,
  DateField,
  NumberField,
  SingleFieldList,
  TextField,
  useRecordContext,
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

const AlbumDetails = (props) => {
  const classes = useStyles()
  const translate = useTranslate()
  const record = useRecordContext(props)
  const data = {
    albumArtist: <TextField source={'albumArtist'} />,
    genre: (
      <ArrayField source={'genres'}>
        <SingleFieldList linkType={false}>
          <ChipField source={'name'} />
        </SingleFieldList>
      </ArrayField>
    ),
    compilation: <BooleanField source={'compilation'} />,
    updatedAt: <DateField source={'updatedAt'} showTime />,
    comment: <MultiLineTextField source={'comment'} />,
  }

  const optionalFields = ['comment', 'genre']
  optionalFields.forEach((field) => {
    !record[field] && delete data[field]
  })

  return (
    <TableContainer>
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
  )
}

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
    <Datagrid
      expand={<AlbumDetails />}
      rowClick={'show'}
      classes={{ row: classes.row }}
      {...rest}
    >
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
