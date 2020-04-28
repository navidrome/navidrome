import React from 'react'
import { TextField, Show, SimpleShowLayout } from 'react-admin'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import { Title } from '../common'

const TranscodingTitle = ({ record }) => {
  return <Title subTitle={`Transcoding ${record ? record.name : ''}`} />
}

const TranscodingShow = (props) => (
  <>
    <Card>
      <CardContent>
        <Typography>
          <Box fontWeight="fontWeightBold" component={'span'}>
            NOTE:
          </Box>{' '}
          Changing the transcoding configuration through the web interface is
          disabled for security reasons. If you would like to change (edit or
          add) transcoding options, restart the server with the{' '}
          <Box fontFamily="Monospace" component={'span'}>
            ND_ENABLETRANSCODINGCONFIG=true
          </Box>{' '}
          configuration option.
        </Typography>
      </CardContent>
    </Card>

    <Show title={<TranscodingTitle />} {...props}>
      <SimpleShowLayout>
        <TextField source="name" />
        <TextField source="targetFormat" />
        <TextField source="defaultBitRate" />
        <TextField source="command" />
      </SimpleShowLayout>
    </Show>
  </>
)

export default TranscodingShow
