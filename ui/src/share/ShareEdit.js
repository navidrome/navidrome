import {
  DateTimeInput,
  BooleanInput,
  Edit,
  NumberField,
  SimpleForm,
  TextInput,
} from 'react-admin'
import { shareUrl } from '../utils'
import { Link } from '@material-ui/core'
import { DateField } from '../common'
import config from '../config'

export const ShareEdit = (props) => {
  const { id, basePath, hasCreate, ...rest } = props
  const url = shareUrl(id)
  return (
    <Edit {...props}>
      <SimpleForm {...rest}>
        <Link source="URL" href={url} target="_blank" rel="noopener noreferrer">
          {url}
        </Link>
        <TextInput source="description" />
        {config.enableDownloads && <BooleanInput source="downloadable" />}
        <DateTimeInput source="expiresAt" />
        <TextInput source="contents" disabled />
        <TextInput source="format" disabled />
        <TextInput source="maxBitRate" disabled />
        <TextInput source="username" disabled />
        <NumberField source="visitCount" disabled />
        <DateField source="lastVisitedAt" disabled showTime />
        <DateField source="createdAt" disabled showTime />
      </SimpleForm>
    </Edit>
  )
}
