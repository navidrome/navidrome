import { DateField, Edit, required, SimpleForm, TextInput } from 'react-admin'
import { urlValidate } from '../utils/validations'
import RadioLinkList from './RadioLinkList'
import RadioPageTitle from './RadioPageTitle'

const RadioEdit = ({ hasCreate, hasEdit, hasList, hasShow, ...props }) => {
  return (
    <Edit title={<RadioPageTitle />} {...props}>
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
        <RadioLinkList />
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

export default RadioEdit
