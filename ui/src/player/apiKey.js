export const API_KEY_PREFIX = 'nav_'
const ALPHABET =
  '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
const KEY_LENGTH = 22

// Bytes >= 248 are dropped so every character is equally likely (248 = 4 * 62).
export const generateApiKey = () => {
  let key = ''
  while (key.length < KEY_LENGTH) {
    for (const b of window.crypto.getRandomValues(new Uint8Array(32))) {
      if (b < 248 && key.length < KEY_LENGTH) key += ALPHABET[b % 62]
    }
  }
  return API_KEY_PREFIX + key
}
