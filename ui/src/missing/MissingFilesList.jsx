import { List, SizeField, useResourceRefresh } from '../common/index'
import {
  Datagrid,
  DateField,
  TextField,
  downloadCSV,
  Pagination,
  Filter,
  ReferenceInput,
  useTranslate,
  SelectInput,
} from 'react-admin'
import jsonExport from 'jsonexport/dist'
import DeleteMissingFilesButton from './DeleteMissingFilesButton.jsx'
import MissingListActions from './MissingListActions.jsx'
import React from 'react'

const exporter = (files) => {
  const filesToExport = files.map((file) => {
    const { path } = file
    return { path }
  })
  jsonExport(filesToExport, { includeHeaders: false }, (err, csv) => {
    downloadCSV(csv, 'navidrome_missing_files')
  })
}

const MissingFilesFilter = (props) => {
  const translate = useTranslate()
  return (
    <Filter {...props} variant={'outlined'}>
      <ReferenceInput
        label={translate('resources.missing.fields.libraryName')}
        source="library_id"
        reference="library"
        sort={{ field: 'name', order: 'ASC' }}
        filterToQuery={(searchText) => ({ name: [searchText] })}
        alwaysOn
      >
        <SelectInput emptyText="-- All --" optionText="name" />
      </ReferenceInput>
    </Filter>
  )
}

const BulkActionButtons = (props) => (
  <>
    <DeleteMissingFilesButton {...props} />
  </>
)

const MissingPagination = (props) => (
  <Pagination rowsPerPageOptions={[50, 100, 200]} {...props} />
)

const MissingFilesList = (props) => {
  useResourceRefresh('song')
  return (
    <List
      {...props}
      sort={{ field: 'updated_at', order: 'DESC' }}
      exporter={exporter}
      actions={<MissingListActions />}
      filters={<MissingFilesFilter />}
      bulkActionButtons={<BulkActionButtons />}
      perPage={50}
      pagination={<MissingPagination />}
    >
      <Datagrid>
        <TextField source={'libraryName'} />
        <TextField source={'path'} />
        <SizeField source={'size'} />
        <DateField source={'updatedAt'} showTime />
      </Datagrid>
    </List>
  )
}

export default MissingFilesList
