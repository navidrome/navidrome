import {
  DateTimeInput,
  BooleanInput,
  Edit,
  NumberField,
  SimpleForm,
  TextInput,
} from 'react-admin'
import { sharePlayerUrl, shareAPlayerUrl } from '../utils'
import { Link, Box, Typography } from '@material-ui/core'
import { DateField } from '../common'
import config from '../config'

export const ShareEdit = (props) => {
  const { id, basePath, hasCreate, ...rest } = props
  const url = sharePlayerUrl(id)
  const aplayerUrl = shareAPlayerUrl(id)
  return (
    <Edit {...props}>
      <SimpleForm {...rest}>
        <Box mb={2}>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            Share URL
          </Typography>
          <Link source="URL" href={url} target="_blank" rel="noopener noreferrer">
            {url}
          </Link>
        </Box>
        <Box mb={2}>
          <Typography variant="body2" color="textSecondary" gutterBottom>
            APlayer Embed URL
          </Typography>
          <Link source="APlayerURL" href={aplayerUrl} target="_blank" rel="noopener noreferrer">
            {aplayerUrl}
          </Link>
        </Box>
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
