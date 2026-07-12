// Timezone-to-country lookup for common IANA zones. Timezone is a much
// stronger signal for physical location than browser/OS language - many
// systems stay on "en-US" regardless of where they actually are, but a
// zone like "Australia/Sydney" is unambiguous.
const TIMEZONE_COUNTRY = {
  'Australia/Sydney': 'AU',
  'Australia/Melbourne': 'AU',
  'Australia/Brisbane': 'AU',
  'Australia/Perth': 'AU',
  'Australia/Adelaide': 'AU',
  'Australia/Darwin': 'AU',
  'Australia/Hobart': 'AU',
  'Australia/Broken_Hill': 'AU',
  'Australia/Lord_Howe': 'AU',
  'Pacific/Auckland': 'NZ',
  'Pacific/Chatham': 'NZ',
  'Europe/London': 'GB',
  'Europe/Belfast': 'GB',
  'Europe/Dublin': 'IE',
  'Europe/Paris': 'FR',
  'Europe/Berlin': 'DE',
  'Europe/Madrid': 'ES',
  'Europe/Rome': 'IT',
  'Europe/Amsterdam': 'NL',
  'Europe/Brussels': 'BE',
  'Europe/Lisbon': 'PT',
  'Europe/Vienna': 'AT',
  'Europe/Zurich': 'CH',
  'Europe/Stockholm': 'SE',
  'Europe/Oslo': 'NO',
  'Europe/Copenhagen': 'DK',
  'Europe/Helsinki': 'FI',
  'Europe/Warsaw': 'PL',
  'Europe/Moscow': 'RU',
  'Asia/Tokyo': 'JP',
  'Asia/Shanghai': 'CN',
  'Asia/Hong_Kong': 'HK',
  'Asia/Kolkata': 'IN',
  'Asia/Calcutta': 'IN',
  'Asia/Singapore': 'SG',
  'Asia/Seoul': 'KR',
  'Asia/Dubai': 'AE',
  'America/New_York': 'US',
  'America/Chicago': 'US',
  'America/Denver': 'US',
  'America/Los_Angeles': 'US',
  'America/Anchorage': 'US',
  'America/Phoenix': 'US',
  'Pacific/Honolulu': 'US',
  'America/Toronto': 'CA',
  'America/Vancouver': 'CA',
  'America/Edmonton': 'CA',
  'America/Winnipeg': 'CA',
  'America/Halifax': 'CA',
  'America/St_Johns': 'CA',
  'America/Mexico_City': 'MX',
  'America/Sao_Paulo': 'BR',
  'America/Argentina/Buenos_Aires': 'AR',
  'Africa/Johannesburg': 'ZA',
}

function fromTimezone() {
  try {
    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone
    if (zone && TIMEZONE_COUNTRY[zone]) {
      return TIMEZONE_COUNTRY[zone]
    }
    // Any Australia/* zone not explicitly listed above is still Australia.
    if (zone && zone.startsWith('Australia/')) {
      return 'AU'
    }
  } catch {
    // ignore, fall through
  }
  return null
}

function fromLocale() {
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
    // ignore, fall through
  }
  return null
}

// Best-effort guess at the user's country, used to request a regionally
// relevant "top podcasts" chart. No IP lookups, no external calls - just
// what the browser already reports about itself. Timezone is checked
// first since it's a much stronger location signal than language/locale.
export default function detectCountry() {
  return fromTimezone() || fromLocale() || 'US'
}
