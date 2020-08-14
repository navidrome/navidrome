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
  const classes = useStyles()
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
            <NumberField source="playCount" sortByOrder={'DESC'} />
            <ArtistContextMenu
              source={'starred'}
              sortBy={'starred ASC, starredAt ASC'}
              sortByOrder={'DESC'}
              label={
                <StarBorderIcon
                  fontSize={'small'}
                  className={classes.columnIcon}
                />
              }
            />
          </ArtistDatagrid>
        )}
      </List>
      <AddToPlaylistDialog />
    </>
  )
}

export default withWidth()(ArtistList)
