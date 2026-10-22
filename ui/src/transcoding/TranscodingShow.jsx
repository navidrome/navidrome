import React from 'react'
import { SelectField, Show, SimpleShowLayout, TextField } from 'react-admin'
import { Title } from '../common'
import { TranscodingNote } from './TranscodingNote'
import { TRANSCODING_BITRATE_CHOICES } from '../consts'

const TranscodingTitle = ({ record }) => {
  return <Title subTitle={`Transcoding ${record ? record.name : ''}`} />
}

const TranscodingShow = (props) => {
  return (
    <>
      <TranscodingNote message={'message.transcodingDisabled'} />

      <Show title={<TranscodingTitle />} {...props}>
        <SimpleShowLayout>
          <TextField source="name" />
          <TextField source="targetFormat" />
          <SelectField
            source="defaultBitRate"
            choices={TRANSCODING_BITRATE_CHOICES}
          />
          <TextField source="command" />
        </SimpleShowLayout>
      </Show>
    </>
  )
}

export default TranscodingShow
