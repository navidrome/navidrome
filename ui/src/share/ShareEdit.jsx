import {
  DateTimeInput,
  BooleanInput,
  Edit,
  NumberField,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { sharePlayerUrl, shareAPlayerUrl } from '../utils'
import { Link, Box, Typography, Divider, Accordion, AccordionSummary, AccordionDetails } from '@material-ui/core'
import { DateField } from '../common'
import config from '../config'
import { EmbedCodeField } from './EmbedCodeField'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

export const ShareEdit = (props) => {
  const { id, basePath, hasCreate, ...rest } = props
  const translate = useTranslate()
  const url = sharePlayerUrl(id)
  const aplayerUrl = shareAPlayerUrl(id)
  return (
    <Edit {...props}>
      <SimpleForm {...rest}>
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
        
        <Accordion>
          <AccordionSummary
            expandIcon={<ExpandMoreIcon />}
            aria-controls="share-urls-content"
            id="share-urls-header"
          >
            <Typography variant="body2" color="textSecondary">
              {translate('message.shareUrl')} & {translate('message.aplayerEmbedUrl')}
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Box mb={2}>
              <Typography variant="body2" color="textSecondary" gutterBottom>
                {translate('message.shareUrl')}
              </Typography>
              <Link href={url} target="_blank" rel="noopener noreferrer">
                {url}
              </Link>
            </Box>
            <Box mb={2}>
              <Typography variant="body2" color="textSecondary" gutterBottom>
                {translate('message.aplayerEmbedUrl')}
              </Typography>
              <Link href={aplayerUrl} target="_blank" rel="noopener noreferrer">
                {aplayerUrl}
              </Link>
            </Box>
            <Box mb={3}>
              <Divider />
            </Box>
            <EmbedCodeField url={aplayerUrl} title={translate('message.navidromeMusicPlayer')} />
          </AccordionDetails>
        </Accordion>
      </SimpleForm>
    </Edit>
  )
}
