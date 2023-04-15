import {
  Create,
  required,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { urlValidate } from '../utils/validations'

const RadioTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.radio.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return <Title subTitle={title} />
}

const RadioCreate = (props) => {
  return (
    <Create title={<RadioTitle />} {...props}>
      <SimpleForm redirect="list" variant={'outlined'}>
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
      </SimpleForm>
    </Create>
  )
}

export default RadioCreate
