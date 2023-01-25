import {
  DateField,
  Edit,
  required,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { urlValidate } from '../utils/validations'
import { Title } from '../common'

const RadioTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.radio.name', {
    smart_count: 1,
  })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const RadioEdit = (props) => {
  return (
    <Edit title={<RadioTitle />} {...props}>
      <SimpleForm variant="outlined" {...props}>
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
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

export default RadioEdit
