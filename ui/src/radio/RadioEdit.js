import { Card, makeStyles } from '@material-ui/core'
import React, { useCallback } from 'react'
import {
  DateField,
  EditContextProvider,
  required,
  SaveButton,
  SimpleForm,
  TextInput,
  Toolbar,
  useEditController,
  useMutation,
  useNotify,
  useRedirect,
} from 'react-admin'
import DeleteRadioButton from './DeleteRadioButton'

const useStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
})

function urlValidate(value) {
  if (!value) {
    return undefined
  }

  try {
    new URL(value)
    return undefined
  } catch (_) {
    return 'ra.validation.url'
  }
}

const RadioToolbar = (props) => (
  <Toolbar {...props} classes={useStyles()}>
    <SaveButton disabled={props.pristine} />
    <DeleteRadioButton />
  </Toolbar>
)

const RadioEditLayout = ({
  hasCreate,
  hasShow,
  hasEdit,
  hasList,
  ...props
}) => {
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()

  const { record } = props

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'radio',
            payload: {
              id: values.id,
              data: {
                name: values.name,
                streamUrl: values.streamUrl,
                homePageUrl: values.homePageUrl,
              },
            },
          },
          { returnPromise: true }
        )
        notify('resources.radio.notifications.updated', 'info', {
          smart_count: 1,
        })
        redirect('/radio')
      } catch (error) {
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect]
  )

  if (!record) {
    return null
  }

  return (
    <>
      {record && (
        <Card>
          <SimpleForm
            variant="outlined"
            save={save}
            toolbar={<RadioToolbar />}
            {...props}
          >
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
        </Card>
      )}
    </>
  )
}

const RadioEdit = (props) => {
  const controllerProps = useEditController(props)
  return (
    <EditContextProvider value={controllerProps}>
      <RadioEditLayout {...props} record={controllerProps.record} />
    </EditContextProvider>
  )
}

export default RadioEdit
