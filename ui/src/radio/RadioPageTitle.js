import { Title, useTranslate } from 'react-admin'

const RadioPageTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.radio.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return <Title subTitle={title} />
}

export default RadioPageTitle
