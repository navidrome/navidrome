import React from 'react'
import { Create, SimpleForm, useTranslate } from 'react-admin'
import { Title } from '../common'
import { playerInputs } from './playerInputs'

const PlayerCreateTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return (
    <Title subTitle={translate('ra.page.create', { name: resourceName })} />
  )
}

const PlayerCreate = (props) => (
  <Create title={<PlayerCreateTitle />} {...props}>
    <SimpleForm variant="outlined" redirect="edit">
      {playerInputs()}
    </SimpleForm>
  </Create>
)

export default PlayerCreate
