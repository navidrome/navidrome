import React from 'react'
import {
  BooleanField,
  Datagrid,
  DateField,
  NumberField,
  Show,
  SimpleShowLayout,
  TextField,
} from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import { makeStyles } from '@material-ui/core/styles'
import {
  ArtistLinkField,
  DurationField,
  RangeField,
  SimpleList,
  SizeField,
  MultiLineTextField,
  AlbumContextMenu,
} from '../common'

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
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
})

const AlbumDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <DateField source="updatedAt" showTime />
        <SizeField source="size" />
        {props.record && props.record['comment'] && (
          <MultiLineTextField source="comment" />
        )}
      </SimpleShowLayout>
    </Show>
  )
}

const AlbumListView = ({ hasShow, hasEdit, hasList, ...rest }) => {
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return isXsmall ? (
    <SimpleList
      primaryText={(r) => r.name}
      secondaryText={(r) => r.albumArtist}
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
      classes={{ row: classes.row }}
      {...rest}
    >
      <TextField source="name" />
      <ArtistLinkField source="artist" />
      {isDesktop && <NumberField source="songCount" sortByOrder={'DESC'} />}
      {isDesktop && <NumberField source="playCount" sortByOrder={'DESC'} />}
      <RangeField source={'year'} sortBy={'maxYear'} sortByOrder={'DESC'} />
      {isDesktop && <DurationField source="duration" />}
      <AlbumContextMenu
        source={'starred'}
        sortBy={'starred ASC, starredAt ASC'}
        sortByOrder={'DESC'}
        className={classes.contextMenu}
        label={
          <StarBorderIcon fontSize={'small'} className={classes.columnIcon} />
        }
      />
    </Datagrid>
  )
}

export default AlbumListView
