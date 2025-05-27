import { makeStyles, useMediaQuery } from '@material-ui/core'
import React, { cloneElement } from 'react'
import {
  CreateButton,
  Datagrid,
  DateField,
  EditButton,
  Filter,
  sanitizeListRestProps,
  SearchInput,
  SimpleList,
  TextField,
  TopToolbar,
  UrlField,
  useTranslate,
} from 'react-admin'
import { List } from '../common'
import { ToggleFieldsMenu, useSelectedFields } from '../common'
import { StreamField } from './StreamField'
import { setTrack } from '../actions'
import { songFromRadio } from './helper'
import { useDispatch } from 'react-redux'

const useStyles = makeStyles({
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

const RadioFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const RadioListActions = ({
  className,
  filters,
  resource,
  showFilter,
  displayedFilters,
  filterValues,
  isAdmin,
  ...rest
}) => {
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))
  const translate = useTranslate()

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      {isAdmin && (
        <CreateButton basePath="/radio">
          {translate('ra.action.create')}
        </CreateButton>
      )}
      {filters &&
        cloneElement(filters, {
          resource,
          showFilter,
          displayedFilters,
          filterValues,
          context: 'button',
        })}
      {isNotSmall && <ToggleFieldsMenu resource="radio" />}
    </TopToolbar>
  )
}

const RadioList = ({ permissions, ...props }) => {
  const classes = useStyles()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  const dispatch = useDispatch()
  const isAdmin = permissions === 'admin'

  const toggleableFields = {
    name: <TextField source="name" />,
    homePageUrl: (
      <UrlField
        source="homePageUrl"
        onClick={(e) => e.stopPropagation()}
        target="_blank"
        rel="noopener noreferrer"
      />
    ),
    streamUrl: <TextField source="streamUrl" />,
    updatedAt: <DateField source="updatedAt" showTime />,
    createdAt: <DateField source="createdAt" showTime />,
  }

  const columns = useSelectedFields({
    resource: 'radio',
    columns: toggleableFields,
    defaultOff: ['createdAt'],
  })

  const handleRowClick = async (id, basePath, record) => {
    dispatch(setTrack(await songFromRadio(record)))
  }

  return (
    <List
      {...props}
      exporter={false}
      sort={{ field: 'name', order: 'ASC' }}
      bulkActionButtons={isAdmin ? undefined : false}
      hasCreate={isAdmin}
      actions={<RadioListActions isAdmin={isAdmin} />}
      filters={<RadioFilter />}
      perPage={isXsmall ? 25 : 10}
    >
      {isXsmall ? (
        <SimpleList
          leftIcon={(r) => (
            <StreamField
              record={r}
              source={'streamUrl'}
              hideUrl
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
              }}
            />
          )}
          primaryText={(r) => r.name}
          secondaryText={(r) => r.homePageUrl}
        />
      ) : (
        <Datagrid rowClick={handleRowClick} classes={{ row: classes.row }}>
          {columns}
          {isAdmin && <EditButton />}
        </Datagrid>
      )}
    </List>
  )
}

export default RadioList
