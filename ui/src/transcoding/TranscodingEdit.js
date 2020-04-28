import React from 'react'
import {
  TextInput,
  SelectInput,
  Edit,
  required,
  SimpleForm,
  useTranslate,
} from 'react-admin'
import { Card, CardContent, Typography, Box } from '@material-ui/core'
import { Title } from '../common'

const TranscodingTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.transcoding.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const TranscodingEdit = (props) => (
  <>
    <Card>
      <CardContent>
        <Typography>
          <Box fontWeight="fontWeightBold" component={'span'}>
            NOTE:
          </Box>{' '}
          Navidrome is currently running with the{' '}
          <Box fontFamily="Monospace" component={'span'}>
            ND_ENABLETRANSCODINGCONFIG=true
          </Box>
          , making it possible to run system commands from the transcoding
          settings using the web interface. We recommend to disable it for
          security reasons and only enable it when configuring Transcoding
          options.
        </Typography>
      </CardContent>
    </Card>
    <Edit title={<TranscodingTitle />} {...props}>
      <SimpleForm>
        <TextInput source="name" validate={[required()]} />
        <TextInput source="targetFormat" validate={[required()]} />
        <SelectInput
          source="defaultBitRate"
          choices={[
            { id: 32, name: '32' },
            { id: 48, name: '48' },
            { id: 64, name: '64' },
            { id: 80, name: '80' },
            { id: 96, name: '96' },
            { id: 112, name: '112' },
            { id: 128, name: '128' },
            { id: 160, name: '160' },
            { id: 192, name: '192' },
            { id: 256, name: '256' },
            { id: 320, name: '320' },
          ]}
        />
        <TextInput source="command" fullWidth validate={[required()]} />
      </SimpleForm>
    </Edit>
  </>
)

export default TranscodingEdit
