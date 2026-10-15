import { useLocale } from 'react-admin'

// Our language codes are mostly region-less ("en"), and Intl reads a bare "en"
// as en-US. Borrow the region from the browser when it speaks the same language.
const resolveDateLocale = (locale, browserLocales = []) => {
  if (!locale || locale.includes('-')) return locale
  const base = locale.toLowerCase()
  return (
    browserLocales.find((l) => l.toLowerCase().split('-')[0] === base) || locale
  )
}

export const useDateLocale = () =>
  resolveDateLocale(useLocale(), navigator.languages)
