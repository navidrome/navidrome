// React Hook to get a list of all languages available. English is hardcoded
import { useGetList } from 'react-admin'

const useGetLanguageChoices = () => {
  const { ids, data, loaded, loading } = useGetList(
    'translation',
    { page: 1, perPage: -1 },
    { field: '', order: '' },
    {},
  )

  const choices = [{ id: 'en', name: 'English' }]
  if (loaded) {
    ids.forEach((id) => choices.push({ id: id, name: data[id].name }))
  }
  choices.sort((a, b) => a.name.localeCompare(b.name))

  return { choices, loaded, loading }
}

export default useGetLanguageChoices
