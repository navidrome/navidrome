import { Card } from '@material-ui/core'
import React from 'react'
import {
  DateField,
  required,
  ShowContextProvider,
  SimpleShowLayout,
  TextField,
  UrlField,
  useShowController,
} from 'react-admin'
import { StreamField } from './StreamField'

const RadioShowLayout = ({ ...props }) => {
  const { record } = props

  if (!record) {
    return null
  }

  return (
    <>
      {record && (
        <Card>
          <SimpleShowLayout>
            <TextField source="name" validate={[required()]} />
            <StreamField source="streamUrl" />
            <UrlField
              type="url"
              source="homePageUrl"
              rel="noreferrer noopener"
              target="_blank"
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
