import { Create, required, SimpleForm, TextInput } from 'react-admin'
import { urlValidate } from '../utils/validations'
import RadioPageTitle from './RadioPageTitle'

const RadioCreate = (props) => {
  return (
    <Create title={<RadioPageTitle />} {...props}>
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
          source="homepageUrl"
          fullWidth
          validate={[urlValidate]}
        />
      </SimpleForm>
    </Create>
  )
}

export default RadioCreate
