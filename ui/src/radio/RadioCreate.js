import { makeStyles } from '@material-ui/core'
import React, { useCallback, useState } from 'react'
import {
  Create,
  required,
  SaveButton,
  SimpleForm,
  TextInput,
  Toolbar,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'
import { urlValidate } from '../utils/validations'
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

const RadioTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.radio.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })
  return <Title subTitle={title} />
}

const RadioCreate = (props) => {
  const styles = useStyles()
  const [favicon, setFavicon] = useState()
  const [loading, setLoading] = useState(false)

  const validateFavicon = useCallback(
    (values) => {
      if (values.favicon && !favicon) {
        return { favicon: 'ra.page.not_found' }
      }
      return undefined
    },
    [favicon]
  )

  return (
    <Create title={<RadioTitle />} {...props}>
      <SimpleForm
        redirect="list"
        variant={'outlined'}
        validate={validateFavicon}
        toolbar={<RadioCreateToolbar isloading={loading.toString()} />}
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
          source="homepageUrl"
          fullWidth
          validate={[urlValidate]}
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
