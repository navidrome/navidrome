import { isDateSet, urlValidate } from './validations'

describe('urlValidate', () => {
  it('returns undefined for valid URLs', () => {
    expect(urlValidate('https://example.com')).toBeUndefined()
    expect(urlValidate('http://localhost:3000')).toBeUndefined()
    expect(urlValidate('ftp://files.example.com')).toBeUndefined()
  })

  it('returns undefined for empty values', () => {
    expect(urlValidate('')).toBeUndefined()
    expect(urlValidate(null)).toBeUndefined()
    expect(urlValidate(undefined)).toBeUndefined()
  })

  it('returns error for invalid URLs', () => {
    expect(urlValidate('not-a-url')).toEqual('ra.validation.url')
    expect(urlValidate('example.com')).toEqual('ra.validation.url')
    expect(urlValidate('://missing-protocol')).toEqual('ra.validation.url')
  })
})

describe('isDateSet', () => {
  describe('with falsy values', () => {
    it('returns false for null', () => {
      expect(isDateSet(null)).toBe(false)
    })

    it('returns false for undefined', () => {
      expect(isDateSet(undefined)).toBe(false)
    })

    it('returns false for empty string', () => {
      expect(isDateSet('')).toBe(false)
    })
  })

  describe('with Go zero date string', () => {
    it('returns false for Go zero date', () => {
      expect(isDateSet('0001-01-01T00:00:00Z')).toBe(false)
    })
  })

  describe('with valid date strings', () => {
    it('returns true for ISO date strings', () => {
      expect(isDateSet('2024-01-15T10:30:00Z')).toBe(true)
      expect(isDateSet('2023-12-25T00:00:00Z')).toBe(true)
    })

    it('returns true for other date formats', () => {
      expect(isDateSet('2024-01-15')).toBe(true)
    })
  })

  describe('with Date objects', () => {
    it('returns true for valid Date objects', () => {
      expect(isDateSet(new Date())).toBe(true)
      expect(isDateSet(new Date('2024-01-15T10:30:00Z'))).toBe(true)
    })

    // Note: Date objects representing Go zero date would return true because
    // toISOString() adds milliseconds (0001-01-01T00:00:00.000Z).
    // In practice, dates from the API come as strings, not Date objects,
    // so this edge case doesn't occur.
  })

  describe('with other truthy values', () => {
    it('returns true for non-date truthy values', () => {
      expect(isDateSet(123)).toBe(true)
      expect(isDateSet({})).toBe(true)
    })
  })
})
