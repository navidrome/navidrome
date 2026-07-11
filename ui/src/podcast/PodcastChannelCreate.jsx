import {
  Create,
  required,
  SimpleForm,
  TextInput,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { urlValidate } from '../utils/validations'

const PodcastChannelTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.podcastChannel.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', { name: `${resourceName}` })
  return <Title subTitle={title} />
}

const PodcastChannelCreate = (props) => {
  return (
    <Create title={<PodcastChannelTitle />} {...props}>
      <SimpleForm redirect="list" variant={'outlined'}>
        <TextInput
          type="url"
          source="url"
          fullWidth
          validate={[required(), urlValidate]}
        />
      </SimpleForm>
    </Create>
  )
}

export default PodcastChannelCreate
