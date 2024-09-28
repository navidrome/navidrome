import React, { useCallback } from 'react'
import { makeStyles } from '@material-ui/core/styles'
import {
  TextInput,
  BooleanInput,
  DateField,
  PasswordInput,
  Edit,
  required,
  email,
  SimpleForm,
  useTranslate,
  Toolbar,
  SaveButton,
  useMutation,
  useNotify,
  useRedirect,
  useRefresh,
  FormDataConsumer,
  usePermissions,
} from 'react-admin'
import { Title } from '../common'
import DeleteUserButton from './DeleteUserButton'

const useStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
})

const UserTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.user.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const UserToolbar = ({ showDelete, ...props }) => (
  <Toolbar {...props} classes={useStyles()}>
    <SaveButton disabled={props.pristine} />
    {showDelete && <DeleteUserButton />}
  </Toolbar>
)

const CurrentPasswordInput = ({ formData, isMyself, ...rest }) => {
  const { permissions } = usePermissions()
  return formData.changePassword && (isMyself || permissions !== 'admin') ? (
    <PasswordInput className="ra-input" source="currentPassword" {...rest} />
  ) : null
}

const NewPasswordInput = ({ formData, ...rest }) => {
  const translate = useTranslate()
  return formData.changePassword ? (
    <PasswordInput
      source="password"
      className="ra-input"
      label={translate('resources.user.fields.newPassword')}
      {...rest}
    />
  ) : null
}

const UserEdit = (props) => {
  const { permissions } = props
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()
  const refresh = useRefresh()

  const isMyself = props.id === localStorage.getItem('userId')
  const getNameHelperText = () =>
    isMyself && {
      helperText: translate('resources.user.helperTexts.name'),
    }
  const canDelete = permissions === 'admin' && !isMyself

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'user',
            payload: { id: values.id, data: values },
          },
          { returnPromise: true },
        )
        notify('resources.user.notifications.updated', 'info', {
          smart_count: 1,
        })
        permissions === 'admin' ? redirect('/user') : refresh()
      } catch (error) {
        if (error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, permissions, redirect, refresh],
  )

  return (
    <Edit title={<UserTitle />} undoable={false} {...props}>
      <SimpleForm
        variant={'outlined'}
        toolbar={<UserToolbar showDelete={canDelete} />}
        save={save}
      >
        {permissions === 'admin' && (
          <TextInput
            spellCheck={false}
            source="userName"
            validate={[required()]}
          />
        )}
        <TextInput
          source="name"
          validate={[required()]}
          {...getNameHelperText()}
        />
        <TextInput spellCheck={false} source="email" validate={[email()]} />
        <BooleanInput source="changePassword" />
        <FormDataConsumer>
          {(formDataProps) => (
            <CurrentPasswordInput
              spellCheck={false}
              isMyself={isMyself}
              {...formDataProps}
            />
          )}
        </FormDataConsumer>
        <FormDataConsumer>
          {(formDataProps) => (
            <NewPasswordInput spellCheck={false} {...formDataProps} />
          )}
        </FormDataConsumer>

        {permissions === 'admin' && (
          <BooleanInput source="isAdmin" initialValue={false} />
        )}
        <DateField variant="body1" source="lastLoginAt" showTime />
        {/*<DateField source="lastAccessAt" showTime />*/}
        <DateField variant="body1" source="updatedAt" showTime />
        <DateField variant="body1" source="createdAt" showTime />
      </SimpleForm>
    </Edit>
  )
}

export default UserEdit
