// Best-effort guess at the user's country from the browser's locale, used
// to request a regionally relevant "top podcasts" chart. No IP lookups, no
// external calls - just what the browser already reports about itself.
export default function detectCountry() {
  try {
    const locale =
      navigator.language ||
      (navigator.languages && navigator.languages[0]) ||
      ''
    const region = locale.split('-')[1]
    if (region && /^[A-Za-z]{2}$/.test(region)) {
      return region.toUpperCase()
    }
  } catch {
    // ignore, fall through to default
  }
  return 'US'
}
