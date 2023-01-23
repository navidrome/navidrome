import { Box, Card, makeStyles } from '@material-ui/core'
import React, { useCallback, useState } from 'react'
import {
  DateField,
  EditContextProvider,
  NumberInput,
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
import { FaviconHandler } from './FaviconHandler'

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

  const [favicon, setFavicon] = useState()
  const [loading, setLoading] = useState(false)

  const save = useCallback(
    async (values) => {
      if (values.favicon && !favicon) {
        return { favicon: 'ra.page.not_found' }
      }

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
    [favicon, mutate, notify, redirect]
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
            <FaviconHandler
              favicon={favicon}
              loading={loading}
              setFavicon={setFavicon}
              setLoading={setLoading}
            />
            <TextInput type="text" source="tags" fullWidth />
            <Box display="flex" width="100% !important">
              <Box flex={1} mr="0.5em">
                <TextInput source="codec" fullWidth variant="outlined" />
              </Box>
              <Box flex={1} mr="0.5em">
                <NumberInput
                  min={0}
                  source="bitrate"
                  fullWidth
                  variant="outlined"
                />
              </Box>
            </Box>
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
