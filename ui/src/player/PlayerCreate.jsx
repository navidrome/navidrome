import React, { useMemo } from 'react'
import { Create, SimpleForm, required, useTranslate } from 'react-admin'
import { Title } from '../common'
import { playerInputs } from './playerInputs'
import ApiKeyInput from './ApiKeyInput'
import { generateApiKey } from './apiKey'

const PlayerCreateTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return (
    <Title subTitle={translate('ra.page.create', { name: resourceName })} />
  )
}

const PlayerCreate = (props) => {
  // Memoized so re-renders don't swap the key the user may have already copied
  const initialValues = useMemo(() => ({ apiKey: generateApiKey() }), [])
  return (
    <Create title={<PlayerCreateTitle />} {...props}>
      <SimpleForm
        variant="outlined"
        redirect="edit"
        initialValues={initialValues}
      >
        {playerInputs()}
        <ApiKeyInput source="apiKey" isCreate validate={required()} />
      </SimpleForm>
    </Create>
  )
}

export default PlayerCreate
