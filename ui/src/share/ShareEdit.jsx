import {
  DateTimeInput,
  BooleanInput,
  Edit,
  SimpleForm,
  TextInput,
} from 'react-admin'
import { sharePlayerUrl } from '../utils'
import { Link } from '@material-ui/core'
import {
  ReadOnlyDateField,
  ReadOnlyNumberField,
  ReadOnlyTextField,
} from '../common'
import config from '../config'

export const ShareEdit = (props) => {
  const { id, basePath, hasCreate, ...rest } = props
  const url = sharePlayerUrl(id)
  return (
    <Edit {...props}>
      <SimpleForm variant={'outlined'} {...rest}>
        <Link
          source="URL"
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          variant="inherit"
        >
          {url}
        </Link>
        <TextInput source="description" />
        {config.enableDownloads && <BooleanInput source="downloadable" />}
        <DateTimeInput source="expiresAt" />
        <ReadOnlyTextField source="contents" />
        <ReadOnlyTextField source="format" />
        <ReadOnlyTextField source="maxBitRate" />
        <ReadOnlyTextField source="username" />
        <ReadOnlyNumberField source="visitCount" />
        <ReadOnlyDateField source="lastVisitedAt" />
        <ReadOnlyDateField source="createdAt" />
      </SimpleForm>
    </Edit>
  )
}
