import React, { useEffect, useState } from 'react'
import {
  AutocompleteInput,
  Create,
  SimpleForm,
  required,
  useDataProvider,
  useNotify,
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

const asChoice = (name) => ({ id: name, name })

const GenreAliasCreate = (props) => {
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const [genreChoices, setGenreChoices] = useState([])

  useEffect(() => {
    dataProvider
      .getList('genre', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'name', order: 'ASC' },
        filter: {},
      })
      .then(({ data }) => setGenreChoices(data.map((g) => asChoice(g.name))))
      .catch(() => notify('ra.page.error', { type: 'warning' }))
  }, [dataProvider, notify])

  const handleCreate = (name) => {
    const choice = asChoice(name)
    setGenreChoices((prev) => [...prev, choice])
    return choice
  }

  return (
    <Create title={<GenreAliasTitle />} {...props}>
      <SimpleForm redirect="list" variant={'outlined'}>
        <AutocompleteInput
          source="aliasName"
          choices={genreChoices}
          optionText="name"
          optionValue="name"
          validate={[required()]}
          onCreate={handleCreate}
          fullWidth
        />
        <AutocompleteInput
          source="canonicalName"
          choices={genreChoices}
          optionText="name"
          optionValue="name"
          validate={[required()]}
          onCreate={handleCreate}
          fullWidth
        />
      </SimpleForm>
    </Create>
  )
}

export default GenreAliasCreate
