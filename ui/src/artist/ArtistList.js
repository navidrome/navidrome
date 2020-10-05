import React, { cloneElement, isValidElement, useState } from 'react'
import { useHistory } from 'react-router-dom'
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
import StarBorderIcon from '@material-ui/icons/StarBorder'
import AddToPlaylistDialog from '../dialogs/AddToPlaylistDialog'
import {
  ArtistContextMenu,
  List,
  QuickFilter,
  SimpleList,
  useGetHandleArtistClick,
} from '../common'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  columnIcon: {
    marginLeft: '3px',
    marginTop: '-2px',
    verticalAlign: 'text-top',
  },
})

const ArtistFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
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

const ArtistListView = ({ hasShow, hasEdit, hasList, width, ...rest }) => {
  const classes = useStyles()
  const handleArtistLink = useGetHandleArtistClick(width)
  const history = useHistory()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return isXsmall ? (
    <SimpleList
      primaryText={(r) => r.name}
      linkType={(id) => {
        history.push(handleArtistLink(id))
      }}
      rightIcon={(r) => <ArtistContextMenu record={r} />}
      {...rest}
    />
  ) : (
    <ArtistDatagrid rowClick={handleArtistLink}>
      <TextField source="name" />
      <NumberField source="albumCount" sortByOrder={'DESC'} />
      <NumberField source="songCount" sortByOrder={'DESC'} />
      <NumberField source="playCount" sortByOrder={'DESC'} />
      <ArtistContextMenu
        source={'starred'}
        sortBy={'starred ASC, starredAt ASC'}
        sortByOrder={'DESC'}
        label={
          <StarBorderIcon fontSize={'small'} className={classes.columnIcon} />
        }
      />
    </ArtistDatagrid>
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
