import {
  Datagrid,
  Filter,
  NumberField,
  SearchInput,
  TextField,
} from 'react-admin'
import { useHistory } from 'react-router-dom'
import { List } from '../common'

const GenreFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const GenreList = (props) => {
  const history = useHistory()
  const handleRowClick = (id) => {
    history.push(`/genre/${id}/show`)
    return false
  }

  return (
    <List
      {...props}
      sort={{ field: 'name', order: 'ASC' }}
      exporter={false}
      bulkActionButtons={false}
      filters={<GenreFilter />}
      perPage={50}
    >
      <Datagrid rowClick={handleRowClick}>
        <TextField source="name" />
        <NumberField source="songCount" sortByOrder={'DESC'} />
        <NumberField source="albumCount" sortByOrder={'DESC'} />
      </Datagrid>
    </List>
  )
}

export default GenreList
