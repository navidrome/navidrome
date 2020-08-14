import React, { cloneElement, isValidElement, useState } from 'react'
import {
  Datagrid,
  DatagridBody,
  DatagridRow,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { useMediaQuery, withWidth } from '@material-ui/core'
import StarIcon from '@material-ui/icons/Star'
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'
import {
  ArtistContextMenu,
  List,
  QuickFilter,
  SimpleList,
  useGetHandleArtistClick,
} from '../common'

const ArtistFilter = (props) => (
  <Filter {...props}>
    <SearchInput source="name" alwaysOn />
    <QuickFilter
      source="starred"
      label={<StarIcon fontSize={'small'} />}
      defaultValue={true}
    />
  </Filter>
)

const ArtistDatagridRow = ({ children, ...rest }) => {
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

const ArtistDatagridBody = (props) => (
  <DatagridBody {...props} row={<ArtistDatagridRow />} />
)
const ArtistDatagrid = (props) => (
  <Datagrid {...props} body={<ArtistDatagridBody />} />
)

const ArtistList = ({ width, ...rest }) => {
  const handleArtistLink = useGetHandleArtistClick(width)
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
    <>
      <List
        {...rest}
        sort={{ field: 'name', order: 'ASC' }}
        exporter={false}
        bulkActionButtons={false}
        filters={<ArtistFilter />}
      >
        {isXsmall ? (
          <SimpleList
            primaryText={(r) => r.name}
            linkType={'show'}
            rightIcon={(r) => <ArtistContextMenu record={r} />}
            {...rest}
          />
        ) : (
          <ArtistDatagrid rowClick={handleArtistLink}>
            <TextField source="name" />
            <NumberField source="albumCount" sortByOrder={'DESC'} />
            <NumberField source="songCount" sortByOrder={'DESC'} />
            <ArtistContextMenu />
          </ArtistDatagrid>
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(ArtistList)
