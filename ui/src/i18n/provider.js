import polyglotI18nProvider from 'ra-i18n-polyglot'
import deepmerge from 'deepmerge'
import dataProvider from '../dataProvider'
import en from './en.json'
import { i18nProvider } from './index'

// Only returns current selected locale if its translations are found in localStorage
const defaultLocale = function () {
  const locale = localStorage.getItem('locale')
  const current = JSON.parse(localStorage.getItem('translation'))
  if (current && current.id === locale) {
    // Asynchronously reload the translation from the server
    retrieveTranslation(locale).then(() => {
      i18nProvider.changeLocale(locale)
    })
    return locale
  }
  return 'en'
}

export function retrieveTranslation(locale) {
  return dataProvider.getOne('translation', { id: locale }).then((res) => {
    localStorage.setItem('translation', JSON.stringify(res.data))
    return prepareLanguage(JSON.parse(res.data.data))
  })
}

const removeEmpty = (obj) => {
  for (let k in obj) {
    if (
      Object.prototype.hasOwnProperty.call(obj, k) &&
      typeof obj[k] === 'object'
    ) {
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
  // Aliases below go on the merged copy: mutating `en` would corrupt the completion baseline
  const merged = deepmerge(en, lang)
  // Make "albumSong" and "playlistTrack" resource use the same translations as "song"
  merged.resources.albumSong = merged.resources.song
  merged.resources.playlistTrack = merged.resources.song
  // ra.boolean.null should always be empty
  merged.ra.boolean.null = ''
  return merged
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
  return retrieveTranslation(locale)
}, defaultLocale())
