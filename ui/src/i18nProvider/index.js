import polyglotI18nProvider from 'ra-i18n-polyglot'
import { useGetList } from 'react-admin'
import deepmerge from 'deepmerge'
import dataProvider from '../dataProvider'
import en from './en.json'

// Only returns current selected locale if its translations are found in localStorage
const defaultLocale = function () {
  const locale = localStorage.getItem('locale')
  const current = localStorage.getItem('translation')
  if (current && current.id === locale) {
    return locale
  }
  return 'en'
}

const prepareLanguage = (lang) => {
  // Make "albumSongs" resource use the same translations as "song"
  lang.resources.albumSong = lang.resources.song
  // ra.boolean.null should always be empty
  lang.ra.boolean.null = ''
  // Fallback to english translations
  return deepmerge(en, lang)
}

const i18nProvider = polyglotI18nProvider((locale) => {
  // English is bundled
  if (locale === 'en') {
    return prepareLanguage(en)
  }
  // If the requested locale is in already loaded, return it
  const current = JSON.parse(localStorage.getItem('translation'))
  if (current && current.id === locale) {
    return prepareLanguage(JSON.parse(current.data))
  }
  // If not, get it from the server, and store it in localStorage
  return dataProvider.getOne('translation', { id: locale }).then((res) => {
    localStorage.setItem('translation', JSON.stringify(res.data))
    return prepareLanguage(JSON.parse(res.data.data))
  })
}, defaultLocale())

export default i18nProvider

// React Hook to get a list of all languages available. English is hardcoded
export const useGetLanguageChoices = () => {
  const { ids, data, loaded, loading } = useGetList(
    'translation',
    { page: 1, perPage: -1 },
    { field: '', order: '' },
    {}
  )

  const choices = [{ id: 'en', name: 'English' }]
  if (loaded) {
    ids.forEach((id) => choices.push({ id: id, name: data[id].name }))
  }
  choices.sort((a, b) => a.name.localeCompare(b.name))

  return { choices, loaded, loading }
}
