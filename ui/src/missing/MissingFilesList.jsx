import { List, SizeField, useResourceRefresh } from '../common/index'
import {
  Datagrid,
  DateField,
  TextField,
  downloadCSV,
  Pagination,
} from 'react-admin'
import jsonExport from 'jsonexport/dist'
import DeleteMissingFilesButton from './DeleteMissingFilesButton.jsx'
import MissingListActions from './MissingListActions.jsx'

const exporter = (files) => {
  const filesToExport = files.map((file) => {
    const { path } = file
    return { path }
  })
  jsonExport(filesToExport, { includeHeaders: false }, (err, csv) => {
    downloadCSV(csv, 'navidrome_missing_files')
  })
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
      bulkActionButtons={<BulkActionButtons />}
      perPage={50}
      pagination={<MissingPagination />}
    >
      <Datagrid>
        <TextField source={'path'} />
        <SizeField source={'size'} />
        <DateField source={'updatedAt'} showTime />
      </Datagrid>
    </List>
  )
}

export default MissingFilesList
