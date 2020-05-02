import polyglotI18nProvider from 'ra-i18n-polyglot'
import deepmerge from 'deepmerge'
import dataProvider from '../dataProvider'
import en from './en.json'

// Only returns current selected locale if its translations are found in localStorage
const defaultLocale = function () {
  const locale = localStorage.getItem('locale')
  const current = JSON.parse(localStorage.getItem('translation'))
  if (current && current.id === locale) {
    return locale
  }
  return 'en'
}

const removeEmpty = (obj) => {
  for (let k in obj) {
    if (obj.hasOwnProperty(k) && typeof obj[k] === 'object') {
      removeEmpty(obj[k])
    } else {
      if (!obj[k]) {
        delete obj[k]
      }
    }
  }
}

const prepareLanguage = (lang) => {
  removeEmpty(lang)
  // Make "albumSongs" resource use the same translations as "song"
  lang.resources.albumSong = lang.resources.song
  // ra.boolean.null should always be empty
  lang.ra.boolean.null = ''
  // Fallback to english translations
  return deepmerge(en, lang)
}

export default polyglotI18nProvider((locale) => {
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
