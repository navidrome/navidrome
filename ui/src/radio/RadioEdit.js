import React, { useCallback, useState } from 'react'
import {
  DateField,
  Edit,
  required,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { urlValidate } from '../utils/validations'
import { FaviconHandler } from './FaviconHandler'
import { QualityRow } from './QualityRow'

const RadioTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.radios.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const RadioEdit = (props) => {
  const [favicon, setFavicon] = useState()
  const [loading, setLoading] = useState(false)

  const validateFavicon = useCallback(
    (values) => {
      if (values.favicon && !favicon) {
        return { favicon: 'ra.page.not_found' }
      }
      return undefined
    },
    [favicon]
  )

  return (
    <Edit title={<RadioTitle />} {...props}>
      <SimpleForm variant="outlined" validate={validateFavicon}>
        <TextInput source="name" validate={[required()]} />
        <TextInput
          type="url"
          source="streamUrl"
          fullWidth
          validate={[required(), urlValidate]}
        />
        <TextInput
          type="url"
          source="homePageUrl"
          fullWidth
          validate={[urlValidate]}
        />
        <FaviconHandler
          favicon={favicon}
          loading={loading}
          setFavicon={setFavicon}
          setLoading={setLoading}
        />
        <TextInput type="text" source="tags" fullWidth />
        <QualityRow />
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

export default RadioEdit
