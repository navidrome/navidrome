import React from 'react'
import { Datagrid, TextField } from 'react-admin'
import { useMediaQuery } from '@material-ui/core'
import { SimpleList, List } from '../common'

const GenreAliasList = (props) => {
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  return (
    <List {...props} exporter={false}>
      {isXsmall ? (
        <SimpleList
          primaryText={(r) => r.aliasName}
          secondaryText={(r) => `-> ${r.canonicalName}`}
        />
      ) : (
        <Datagrid rowClick={false}>
          <TextField source="aliasName" />
          <TextField source="canonicalName" />
        </Datagrid>
      )}
    </List>
  )
}

export default GenreAliasList
