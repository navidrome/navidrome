import React, { useCallback } from 'react'
import {
  BooleanInput,
  Create,
  email,
  FormDataConsumer,
  PasswordInput,
  required,
  SimpleForm,
  TextInput,
  useMutation,
  useNotify,
  useRedirect,
  useTranslate,
} from 'react-admin'
import { Typography } from '@material-ui/core'
import { Title } from '../common'
import { LibrarySelectionField } from './LibrarySelectionField.jsx'

const UserCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const resourceName = translate('resources.user.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'create',
            resource: 'user',
            payload: { data: values },
          },
          { returnPromise: true },
        )
        notify('resources.user.notifications.created', 'info', {
          smart_count: 1,
        })
        redirect('/user')
      } catch (error) {
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect],
  )

  // Custom validation function
  const validateUserForm = (values) => {
    const errors = {}
    // Only require library selection for non-admin users
    if (
      !values.isAdmin &&
      (!values.libraryIds || values.libraryIds.length === 0)
    ) {
      errors.libraryIds = translate(
        'resources.user.validation.librariesRequired',
      )
    }
    return errors
  }

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm save={save} validate={validateUserForm} variant={'outlined'}>
        <TextInput
          spellCheck={false}
          source="userName"
          validate={[required()]}
        />
        <TextInput source="name" validate={[required()]} />
        <TextInput spellCheck={false} source="email" validate={[email()]} />
        <PasswordInput
          spellCheck={false}
          source="password"
          validate={[required()]}
        />
        <BooleanInput source="isAdmin" defaultValue={false} />

        {/* Conditional Library Selection */}
        <FormDataConsumer>
          {({ formData }) => (
            <>
              {!formData.isAdmin && <LibrarySelectionField />}

              {formData.isAdmin && (
                <Typography
                  variant="body2"
                  color="textSecondary"
                  style={{ marginTop: 16, marginBottom: 16 }}
                >
                  {translate('resources.user.messages.adminAutoLibraries')}
                </Typography>
              )}
            </>
          )}
        </FormDataConsumer>
      </SimpleForm>
    </Create>
  )
}

export default UserCreate
