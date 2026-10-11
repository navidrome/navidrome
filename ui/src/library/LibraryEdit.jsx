import React, { useCallback, useState } from 'react'
import PropTypes from 'prop-types'
import {
  Edit,
  FormWithRedirect,
  TextInput,
  BooleanInput,
  Confirm,
  required,
  SaveButton,
  useTranslate,
  useMutation,
  useNotify,
  useRedirect,
  Toolbar,
} from 'react-admin'
import { Typography, Box } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import DeleteLibraryButton from './DeleteLibraryButton'
import {
  ReadOnlyDateField,
  ReadOnlyDurationField,
  ReadOnlyNumberField,
  ReadOnlySizeField,
  Title,
} from '../common'
import config from '../config'
import { PIDInputs } from './PIDInput'
import { pidConfigChanged } from './pidPresets'

const useStyles = makeStyles({
  toolbar: {
    display: 'flex',
    justifyContent: 'space-between',
  },
})

const readOnlyProps = { resource: 'library', fullWidth: true }

const LibraryTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.library.name', { smart_count: 1 })
  return (
    <Title subTitle={`${resourceName} ${record ? `"${record.name}"` : ''}`} />
  )
}

const CustomToolbar = ({ showDelete, ...props }) => (
  <Toolbar {...props} classes={useStyles()}>
    <SaveButton disabled={props.pristine} />
    {showDelete && (
      <DeleteLibraryButton
        record={props.record}
        resource="library"
        basePath="/library"
      />
    )}
  </Toolbar>
)

export const LibraryEditForm = ({ formProps, canEditPath, canDelete }) => {
  const translate = useTranslate()
  const [confirmOpen, setConfirmOpen] = useState(false)

  // Every submit path (Save button and Enter key) goes through here, so a PID change always asks first
  const submit = () => {
    if (
      pidConfigChanged(
        formProps.form.getState().values,
        formProps.record,
        config,
      )
    ) {
      setConfirmOpen(true)
      return
    }
    formProps.handleSubmit()
  }

  const handleConfirm = () => {
    setConfirmOpen(false)
    formProps.handleSubmit()
  }

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        submit()
      }}
    >
      <Box p="1em" maxWidth="800px">
        <Box display="flex">
          <Box flex={1} mr="1em">
            {/* Basic Information */}
            <Typography variant="h6" gutterBottom>
              {translate('resources.library.sections.basic')}
            </Typography>

            <TextInput
              source="name"
              label={translate('resources.library.fields.name')}
              validate={[required()]}
              variant="outlined"
            />
            <TextInput
              source="path"
              label={translate('resources.library.fields.path')}
              validate={[required()]}
              fullWidth
              variant="outlined"
              InputProps={{ readOnly: !canEditPath }} // Disable editing path for library 1
            />
            <BooleanInput
              source="defaultNewUsers"
              label={translate('resources.library.fields.defaultNewUsers')}
              variant="outlined"
            />

            <Box mt="2em" />
            <Typography variant="h6" gutterBottom>
              {translate('resources.library.sections.pid')}
            </Typography>
            <PIDInputs />

            <Box mt="2em" />

            {/* Statistics - Two Column Layout */}
            <Typography variant="h6" gutterBottom>
              {translate('resources.library.sections.statistics')}
            </Typography>

            <Box
              display="grid"
              gridTemplateColumns="1fr 1fr"
              gridColumnGap="1em"
            >
              <ReadOnlyNumberField source="totalSongs" {...readOnlyProps} />
              <ReadOnlyNumberField source="totalAlbums" {...readOnlyProps} />
              <ReadOnlyNumberField source="totalArtists" {...readOnlyProps} />
              <ReadOnlySizeField source="totalSize" {...readOnlyProps} />
              <ReadOnlyDurationField
                source="totalDuration"
                {...readOnlyProps}
              />
              <ReadOnlyNumberField
                source="totalMissingFiles"
                {...readOnlyProps}
              />
              <Box gridColumn="1 / -1">
                <ReadOnlyDateField source="lastScanAt" {...readOnlyProps} />
              </Box>
              <ReadOnlyDateField source="updatedAt" {...readOnlyProps} />
              <ReadOnlyDateField source="createdAt" {...readOnlyProps} />
            </Box>
          </Box>
        </Box>
      </Box>

      <CustomToolbar
        handleSubmitWithRedirect={submit}
        pristine={formProps.pristine}
        saving={formProps.saving}
        record={formProps.record}
        showDelete={canDelete}
      />
      <Confirm
        isOpen={confirmOpen}
        loading={formProps.saving}
        title="resources.library.messages.pidChangeTitle"
        content="resources.library.messages.pidChangeConfirm"
        onConfirm={handleConfirm}
        onClose={() => setConfirmOpen(false)}
      />
    </form>
  )
}

LibraryEditForm.propTypes = {
  formProps: PropTypes.object.isRequired,
  canEditPath: PropTypes.bool,
  canDelete: PropTypes.bool,
}

const LibraryEdit = (props) => {
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()

  // Library ID 1 is protected (main library)
  const canDelete = props.id !== '1'
  const canEditPath = props.id !== '1'

  const save = useCallback(
    async (values) => {
      try {
        await mutate(
          {
            type: 'update',
            resource: 'library',
            payload: { id: values.id, data: values },
          },
          { returnPromise: true },
        )
        notify('resources.library.notifications.updated', 'info', {
          smart_count: 1,
        })
        redirect('/library')
      } catch (error) {
        if (error.body && error.body.errors) {
          return error.body.errors
        }
      }
    },
    [mutate, notify, redirect],
  )

  return (
    <Edit title={<LibraryTitle />} undoable={false} {...props}>
      <FormWithRedirect
        {...props}
        save={save}
        render={(formProps) => (
          <LibraryEditForm
            formProps={formProps}
            canEditPath={canEditPath}
            canDelete={canDelete}
          />
        )}
      />
    </Edit>
  )
}

export default LibraryEdit
