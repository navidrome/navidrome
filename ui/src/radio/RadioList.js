import { makeStyles, useMediaQuery } from '@material-ui/core'
import React, { cloneElement } from 'react'
import {
  CreateButton,
  Datagrid,
  DateField,
  List,
  sanitizeListRestProps,
  TextField,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import { ToggleFieldsMenu, useSelectedFields } from '../common'
import { RadioContextMenu } from './RadioContextMenu'

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

  const isAdmin = permissions === 'admin'

  const toggleableFields = {
    name: <TextField source="name" />,
    streamUrl: <TextField source="streamUrl" />,
    homePageUrl: <TextField source="homePageUrl" />,
    createdAt: <DateField source="createdAt" showTime />,
    updatedAt: <DateField source="updatedAt" showTime />,
  }

  const columns = useSelectedFields({
    resource: 'radio',
    columns: toggleableFields,
    defaultOff: ['homePageUrl', 'createdAt', 'updatedAt'],
  })

  return (
    <List
      {...props}
      exporter={false}
      bulkActionButtons={isAdmin ? undefined : false}
      hasCreate={isAdmin}
      actions={<RadioListActions isAdmin={isAdmin} />}
    >
      <Datagrid
        rowClick={isAdmin ? 'edit' : 'show'}
        classes={{ row: classes.row }}
      >
        {columns}
        <RadioContextMenu className={classes.contextMenu} />
      </Datagrid>
    </List>
  )
}

export default RadioList
