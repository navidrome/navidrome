import {
  DateField,
  DateInput,
  Edit,
  NumberField,
  SimpleForm,
  TextInput,
} from 'react-admin'
import { shareUrl } from '../utils'
import { Link } from '@material-ui/core'

export const ShareEdit = (props) => {
  const { id } = props
  const url = shareUrl(id)
  return (
    <Edit {...props}>
      <SimpleForm>
        <Link source="URL" href={url} target="_blank" rel="noopener noreferrer">
          {url}
        </Link>
        <TextInput source="description" />
        <TextInput source="contents" disabled />
        <TextInput source="format" disabled />
        <TextInput source="maxBitRate" disabled />
        <DateInput source="expiresAt" disabled />
        <TextInput source="username" disabled />
        <NumberField source="visitCount" disabled />
        <DateField source="lastVisitedAt" disabled />
        <DateField source="createdAt" disabled />
      </SimpleForm>
    </Edit>
  )
}
