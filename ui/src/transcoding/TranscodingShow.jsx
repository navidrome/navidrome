import React from 'react'
import { Show, SimpleShowLayout, TextField } from 'react-admin'
import { Title } from '../common'
import { TranscodingNote } from './TranscodingNote'

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
          <TextField source="defaultBitRate" />
          <TextField source="command" />
        </SimpleShowLayout>
      </Show>
    </>
  )
}

export default TranscodingShow
