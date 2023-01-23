import { makeStyles } from '@material-ui/core'
import React, { useCallback, useState } from 'react'
import {
  Create,
  required,
  SaveButton,
  SimpleForm,
  TextInput,
  Toolbar,
  useMutation,
  useNotify,
  useRedirect,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { FaviconHandler } from './FaviconHandler'
import { QualityRow } from './QualityRow'

const useStyles = makeStyles({
  hidden: {
    display: 'none !important',
  },
})

const RadioCreateToolbar = (props) => (
  <Toolbar {...props}>
    <SaveButton disabled={props.invalid || props.isloading === 'true'} />
  </Toolbar>
)

const RadioCreate = (props) => {
  const translate = useTranslate()
  const [mutate] = useMutation()
  const notify = useNotify()
  const redirect = useRedirect()

  const styles = useStyles()

  const resourceName = translate('resources.radio.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
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
            type: 'create',
            resource: 'radio',
            payload: { data: values },
          },
          { returnPromise: true }
        )
        notify('resources.radio.notifications.created', 'info', {
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

  return (
    <Create title={<Title subTitle={title} />} {...props}>
      <SimpleForm
        save={save}
        variant="outlined"
        toolbar={<RadioCreateToolbar isloading={loading.toString()} />}
      >
        <TextInput source="name" validate={[required()]} fullWidth />
        <TextInput
          type="url"
          source="streamUrl"
          fullWidth
          validate={[required()]}
        />
        <TextInput type="url" source="homepageUrl" fullWidth />
        <FaviconHandler
          favicon={favicon}
          loading={loading}
          setFavicon={setFavicon}
          setLoading={setLoading}
        />
        <TextInput source="tags" fullWidth />
        <QualityRow />
        <TextInput
          className={styles.hidden}
          source="radioInfoId"
          disabled
          hidden
        />
      </SimpleForm>
    </Create>
  )
}

export default RadioCreate
