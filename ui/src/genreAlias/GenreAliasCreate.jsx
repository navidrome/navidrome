import React, { useEffect, useState } from 'react'
import { Card, CardContent, Typography, Button } from '@material-ui/core'
import { TextField as MuiTextField } from '@material-ui/core'
import Autocomplete from '@material-ui/lab/Autocomplete'
import {
  Create,
  useDataProvider,
  useNotify,
  useRedirect,
  useTranslate,
} from 'react-admin'
import { Title } from '../common'

const GenreAliasTitle = () => {
  const translate = useTranslate()
  const resourceName = translate('resources.genreAlias.name', {
    smart_count: 1,
  })
  const title = translate('ra.page.create', { name: `${resourceName}` })
  return <Title subTitle={title} />
}

const MergeGenresForm = () => {
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const redirect = useRedirect()

  const [genreNames, setGenreNames] = useState([])
  const [sources, setSources] = useState([])
  const [target, setTarget] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    dataProvider
      .getList('genre', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'name', order: 'ASC' },
        filter: {},
      })
      .then(({ data }) => setGenreNames(data.map((g) => g.name)))
      .catch(() => notify('ra.page.error', { type: 'warning' }))
  }, [dataProvider, notify])

  const trimmedTarget = target.trim()
  const canSubmit =
    sources.length > 0 &&
    trimmedTarget !== '' &&
    !sources.includes(trimmedTarget)

  const handleSubmit = async () => {
    setSubmitting(true)
    const results = await Promise.allSettled(
      sources.map((aliasName) =>
        dataProvider.create('genreAlias', {
          data: { aliasName, canonicalName: trimmedTarget },
        }),
      ),
    )
    setSubmitting(false)
    const failedSources = sources.filter(
      (_, idx) => results[idx].status === 'rejected',
    )
    const succeededCount = sources.length - failedSources.length

    if (failedSources.length === 0) {
      notify('resources.genreAlias.message.mergeSuccess', {
        messageArgs: { smart_count: succeededCount, target: trimmedTarget },
      })
      redirect('list', '/genreAlias')
    } else {
      notify('resources.genreAlias.message.mergePartialFailure', {
        type: 'warning',
        messageArgs: {
          failed: failedSources.length,
          names: failedSources.join(', '),
        },
      })
      setSources(failedSources)
    }
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="body2" gutterBottom>
          {translate('resources.genreAlias.mergeHelp')}
        </Typography>
        <Autocomplete
          multiple
          options={genreNames}
          value={sources}
          onChange={(event, value) => setSources(value)}
          renderInput={(params) => (
            <MuiTextField
              {...params}
              variant="outlined"
              margin="normal"
              label={translate('resources.genreAlias.mergeSources')}
              placeholder={translate('resources.genreAlias.selectSources')}
            />
          )}
        />
        <Autocomplete
          freeSolo
          options={genreNames}
          inputValue={target}
          onInputChange={(event, value) => setTarget(value)}
          onChange={(event, value) => setTarget(value || '')}
          renderInput={(params) => (
            <MuiTextField
              {...params}
              variant="outlined"
              margin="normal"
              label={translate('resources.genreAlias.mergeTarget')}
              helperText={translate('resources.genreAlias.targetHelp')}
            />
          )}
        />
        <Button
          variant="contained"
          color="primary"
          disabled={!canSubmit || submitting}
          onClick={handleSubmit}
        >
          {translate('resources.genreAlias.merge')}
        </Button>
      </CardContent>
    </Card>
  )
}

const GenreAliasCreate = (props) => (
  <Create title={<GenreAliasTitle />} {...props}>
    <MergeGenresForm />
  </Create>
)

export default GenreAliasCreate
