import { useLocale } from 'react-admin'

export const getStoredDateFormat = () =>
  localStorage.getItem('dateFormat') || 'language'

// Our language codes are mostly region-less ("en"), and Intl reads a bare "en"
// as en-US. Borrow the region from the browser when it speaks the same language.
const resolveDateLocale = (locale, browserLocales = []) => {
  if (!locale || locale.includes('-')) return locale
  const base = locale.toLowerCase()
  return (
    browserLocales.find((l) => l.toLowerCase().split('-')[0] === base) || locale
  )
}

// With the "browser" date format, return undefined so Intl uses the browser's
// own locale, regardless of the selected language.
export const useDateLocale = () => {
  const locale = useLocale()
  if (getStoredDateFormat() === 'browser') return undefined
  return resolveDateLocale(locale, navigator.languages)
}
