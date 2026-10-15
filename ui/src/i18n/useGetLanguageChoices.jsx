// React Hook to get a list of all languages available. English is hardcoded
import { useGetList } from 'react-admin'
import en from './en.json'

const countLeaves = (obj) =>
  Object.values(obj).reduce(
    (sum, v) =>
      sum + (typeof v === 'object' && v !== null ? countLeaves(v) : v ? 1 : 0),
    0,
  )

const enTermCount = countLeaves(en)

const withPercentage = ({ id, name, termCount }) => {
  if (!termCount) return { id, name }
  const pct = Math.min(100, Math.round((100 * termCount) / enTermCount))
  // Isolate the percentage, or it renders as "(%61)" next to a right-to-left name
  return { id, name: `${name} ⁦(${pct}%)⁩` }
}

const useGetLanguageChoices = () => {
  const { ids, data, loaded, loading } = useGetList(
    'translation',
    { page: 1, perPage: -1 },
    { field: '', order: '' },
    {},
  )

  const languages = [{ id: 'en', name: 'English', termCount: enTermCount }]
  if (loaded) {
    ids.forEach((id) =>
      languages.push({
        id,
        name: data[id].name,
        termCount: data[id].termCount,
      }),
    )
  }
  languages.sort((a, b) => a.name.localeCompare(b.name))

  return { choices: languages.map(withPercentage), loaded, loading }
}

export default useGetLanguageChoices
