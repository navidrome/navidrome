import {
  DateField,
  Show,
  SimpleShowLayout,
  TextField,
  UrlField,
  useTranslate,
} from 'react-admin'
import RadioLinkList from './RadioLinkList'

const RadioTitle = ({ record }) => {
  const translate = useTranslate()

  return (
    <span>
      {translate('resources.radio.name', { smart_count: 1 })} {record.name}
    </span>
  )
}

const RadioShow = (props) => {
  return (
    <Show title={<RadioTitle />} {...props}>
      <SimpleShowLayout>
        <TextField source="name" />
        <TextField source="streamUrl" />
        <UrlField
          source="homePageUrl"
          onClick={(e) => e.stopPropagation()}
          target="_blank"
          rel="noopener noreferrer"
        />{' '}
        <RadioLinkList />
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleShowLayout>
    </Show>
  )
}

export default RadioShow
