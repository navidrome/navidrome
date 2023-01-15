import { makeStyles, useMediaQuery } from '@material-ui/core'
import React, { cloneElement } from 'react'
import {
  CreateButton,
  Datagrid,
  DateField,
  Filter,
  List,
  sanitizeListRestProps,
  SearchInput,
  SimpleList,
  TextField,
  TopToolbar,
  UrlField,
  useTranslate,
} from 'react-admin'
import { ToggleFieldsMenu, useSelectedFields } from '../common'
import { StreamField } from './StreamField'

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
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))

  const classes = useStyles()

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
    streamUrl: <StreamField source="streamUrl" />,
    createdAt: <DateField source="createdAt" showTime />,
    updatedAt: <DateField source="updatedAt" showTime />,
  }

  const columns = useSelectedFields({
    resource: 'radio',
    columns: toggleableFields,
    defaultOff: ['updatedAt'],
  })

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
          linkType={isAdmin ? 'edit' : 'show'}
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
        <Datagrid
          rowClick={isAdmin ? 'edit' : 'show'}
          classes={{ row: classes.row }}
        >
          {columns}
        </Datagrid>
      )}
    </List>
  )
}

export default RadioList
