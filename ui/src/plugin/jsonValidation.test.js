import { describe, it, expect } from 'vitest'
import { validateJson, formatJson } from './jsonValidation'

describe('validateJson', () => {
  it('returns valid for empty string', () => {
    const result = validateJson('')
    expect(result.valid).toBe(true)
    expect(result.error).toBeNull()
    expect(result.parsed).toBeNull()
  })

  it('returns valid for whitespace only', () => {
    const result = validateJson('   ')
    expect(result.valid).toBe(true)
    expect(result.error).toBeNull()
  })

  it('returns valid for valid JSON object', () => {
    const result = validateJson('{"key": "value"}')
    expect(result.valid).toBe(true)
    expect(result.error).toBeNull()
    expect(result.parsed).toEqual({ key: 'value' })
  })

  it('returns valid for nested JSON object', () => {
    const result = validateJson('{"outer": {"inner": 123}}')
    expect(result.valid).toBe(true)
    expect(result.parsed).toEqual({ outer: { inner: 123 } })
  })

  it('returns invalid for JSON array', () => {
    const result = validateJson('[1, 2, 3]')
    expect(result.valid).toBe(false)
    expect(result.error).toBe('Configuration must be a JSON object')
  })

  it('returns invalid for JSON primitive string', () => {
    const result = validateJson('"hello"')
    expect(result.valid).toBe(false)
    expect(result.error).toBe('Configuration must be a JSON object')
  })

  it('returns invalid for JSON primitive number', () => {
    const result = validateJson('42')
    expect(result.valid).toBe(false)
    expect(result.error).toBe('Configuration must be a JSON object')
  })

  it('returns invalid for JSON null', () => {
    const result = validateJson('null')
    expect(result.valid).toBe(false)
    expect(result.error).toBe('Configuration must be a JSON object')
  })

  it('returns invalid for malformed JSON', () => {
    const result = validateJson('{"key": }')
    expect(result.valid).toBe(false)
    expect(result.error).toContain('Invalid JSON')
  })

  it('returns invalid for incomplete JSON', () => {
    const result = validateJson('{"key": "value"')
    expect(result.valid).toBe(false)
    expect(result.error).toContain('Invalid JSON')
  })

  it('returns invalid for JSON with trailing comma', () => {
    const result = validateJson('{"key": "value",}')
    expect(result.valid).toBe(false)
    expect(result.error).toContain('Invalid JSON')
  })
})

describe('formatJson', () => {
  it('returns empty string unchanged', () => {
    expect(formatJson('')).toBe('')
  })

  it('returns whitespace unchanged', () => {
    expect(formatJson('  ')).toBe('  ')
  })

  it('formats compact JSON with indentation', () => {
    const result = formatJson('{"key":"value"}')
    expect(result).toBe('{\n  "key": "value"\n}')
  })

  it('formats nested JSON with proper indentation', () => {
    const result = formatJson('{"outer":{"inner":123}}')
    expect(result).toBe('{\n  "outer": {\n    "inner": 123\n  }\n}')
  })

  it('returns invalid JSON unchanged', () => {
    const invalid = '{"key": }'
    expect(formatJson(invalid)).toBe(invalid)
  })
})
