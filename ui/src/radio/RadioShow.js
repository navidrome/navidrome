import { Card } from '@material-ui/core'
import {
  DateField,
  required,
  ShowContextProvider,
  SimpleShowLayout,
  TextField,
  UrlField,
  useShowController,
} from 'react-admin'
import RadioActions from './RadioActions'

const RadioShowLayout = ({ ...props }) => {
  const { record } = props

  if (!record) {
    return null
  }

  return (
    <>
      {record && <RadioActions record={record} />}
      {record && (
        <Card>
          <SimpleShowLayout>
            <TextField source="name" validate={[required()]} />
            <UrlField type="url" source="streamUrl" rel="noreferrer noopener" />
            <UrlField
              type="url"
              source="homePageUrl"
              rel="noreferrer noopener"
            />
            <DateField variant="body1" source="updatedAt" showTime />
            <DateField variant="body1" source="createdAt" showTime />
          </SimpleShowLayout>
        </Card>
      )}
    </>
  )
}

const RadioShow = (props) => {
  const controllerProps = useShowController(props)
  return (
    <ShowContextProvider value={controllerProps}>
      <RadioShowLayout {...props} record={controllerProps.record} />
    </ShowContextProvider>
  )
}

export default RadioShow
