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
import {
  ArtistLinkField,
  DurationField,
  RangeField,
  SimpleList,
} from '../common'
import { useMediaQuery } from '@material-ui/core'
import AlbumContextMenu from '../common/AlbumContextMenu'

const AlbumDetails = (props) => {
  return (
    <Show {...props} title=" ">
      <SimpleShowLayout>
        <TextField source="albumArtist" />
        <TextField source="genre" />
        <BooleanField source="compilation" />
        <DateField source="updatedAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

const AlbumListView = ({ hasShow, hasEdit, hasList, ...rest }) => {
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return isXsmall ? (
    <SimpleList
      primaryText={(r) => r.name}
      secondaryText={(r) => r.albumArtist}
      tertiaryText={(r) => (
        <>
          <RangeField record={r} source={'year'} sortBy={'maxYear'} />
          &nbsp;&nbsp;&nbsp;
        </>
      )}
      linkType={'show'}
      rightIcon={(r) => <AlbumContextMenu record={r} />}
      {...rest}
    />
  ) : (
    <Datagrid expand={<AlbumDetails />} rowClick={'show'} {...rest}>
      <TextField source="name" />
      <ArtistLinkField />
      {isDesktop && <NumberField source="songCount" />}
      {isDesktop && <NumberField source="playCount" />}
      <RangeField source={'year'} sortBy={'maxYear'} />
      {isDesktop && <DurationField source="duration" />}
      <AlbumContextMenu />
    </Datagrid>
  )
}

export default AlbumListView
