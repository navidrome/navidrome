import polyglotI18nProvider from 'ra-i18n-polyglot'
import dataProvider from '../dataProvider'
import en from './en.json'

const defaultLocale = function () {
  const locale = localStorage.getItem('locale')
  const current = JSON.parse(localStorage.getItem('translation'))
  if (current && current.id === locale) {
    return locale
  }
  return 'en'
}

const i18nProvider = polyglotI18nProvider((locale) => {
  if (locale === 'en') {
    return en
  }
  const current = JSON.parse(localStorage.getItem('translation'))
  if (current && current.id === locale) {
    return JSON.parse(current.data)
  }
  return dataProvider.getOne('translation', { id: locale }).then((res) => {
    localStorage.setItem('translation', JSON.stringify(res.data))
    return JSON.parse(res.data.data)
  })
}, defaultLocale())

export default i18nProvider
