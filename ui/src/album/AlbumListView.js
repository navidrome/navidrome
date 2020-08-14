import React, { cloneElement, isValidElement, useState } from 'react'
import {
  BooleanField,
  Datagrid,
  DatagridBody,
  DatagridRow,
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
import { AlbumContextMenu } from '../common'

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

const AlbumDatagridRow = ({ children, ...rest }) => {
  const [visible, setVisible] = useState(false)
  const childCount = React.Children.count(children)
  return (
    <DatagridRow
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
      {...rest}
    >
      {React.Children.map(
        children,
        (child, index) =>
          child &&
          isValidElement(child) &&
          (index < childCount - 1
            ? child
            : cloneElement(child, {
                visible,
              }))
      )}
    </DatagridRow>
  )
}

const AlbumDatagridBody = (props) => (
  <DatagridBody {...props} row={<AlbumDatagridRow />} />
)
const AlbumDatagrid = (props) => (
  <Datagrid {...props} body={<AlbumDatagridBody />} />
)

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
    <AlbumDatagrid expand={<AlbumDetails />} rowClick={'show'} {...rest}>
      <TextField source="name" />
      <ArtistLinkField source="artist" />
      {isDesktop && <NumberField source="songCount" sortByOrder={'DESC'} />}
      {isDesktop && <NumberField source="playCount" sortByOrder={'DESC'} />}
      <RangeField source={'year'} sortBy={'maxYear'} sortByOrder={'DESC'} />
      {isDesktop && <DurationField source="duration" />}
      <AlbumContextMenu />
    </AlbumDatagrid>
  )
}

export default AlbumListView
