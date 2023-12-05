import PropTypes from 'prop-types'
import {
  ArrayField,
  Datagrid,
  Labeled,
  TextField,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { useDispatch } from 'react-redux'
import { playRadio } from '../actions'
import { songFromRadio } from './helper'

const RadioLinkList = (props) => {
  const record = useRecordContext(props)
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handleRowClick = async (id, basePath, row) => {
    dispatch(playRadio(await songFromRadio(record, row)))
  }

  return record.links ? (
    <Labeled label={translate('resources.radio.fields.links')} fullWidth>
      <ArrayField source="links">
        <Datagrid rowClick={handleRowClick}>
          <TextField source="name" />
          <TextField source="url" />
        </Datagrid>
      </ArrayField>
    </Labeled>
  ) : null
}

RadioLinkList.propTypes = {
  record: PropTypes.object,
}

export default RadioLinkList
