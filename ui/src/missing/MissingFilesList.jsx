import { List, SizeField } from '../common/index.js'
import { Datagrid, DateField, TextField, downloadCSV } from 'react-admin'
import jsonExport from 'jsonexport/dist'

const exporter = (files) => {
  const filesToExport = files.map((file) => {
    const { path } = file
    return { path }
  })
  jsonExport(filesToExport, { includeHeaders: false }, (err, csv) => {
    downloadCSV(csv, 'navidrome_missing_files')
  })
}

const MissingFilesList = (props) => {
  return (
    <List
      {...props}
      filter={{ missing: true }}
      sort={{ field: 'updated_at', order: 'DESC' }}
      exporter={exporter}
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
