import { describe, it, expect } from 'vitest'
import {
  formatTomlValue,
  buildTomlSections,
  configToToml,
  separateAndSortConfigs,
} from './toml'

describe('formatTomlValue', () => {
  it('handles null and undefined values', () => {
    expect(formatTomlValue(null)).toBe('""')
    expect(formatTomlValue(undefined)).toBe('""')
  })

  it('handles boolean values', () => {
    expect(formatTomlValue('true')).toBe('true')
    expect(formatTomlValue('false')).toBe('false')
    expect(formatTomlValue(true)).toBe('true')
    expect(formatTomlValue(false)).toBe('false')
  })

  it('handles integer values', () => {
    expect(formatTomlValue('123')).toBe('123')
    expect(formatTomlValue('-456')).toBe('-456')
    expect(formatTomlValue('0')).toBe('0')
    expect(formatTomlValue(789)).toBe('789')
  })

  it('handles float values', () => {
    expect(formatTomlValue('123.45')).toBe('123.45')
    expect(formatTomlValue('-67.89')).toBe('-67.89')
    expect(formatTomlValue('0.0')).toBe('0.0')
    expect(formatTomlValue(12.34)).toBe('12.34')
  })

  it('handles duration values', () => {
    expect(formatTomlValue('300ms')).toBe('"300ms"')
    expect(formatTomlValue('5s')).toBe('"5s"')
    expect(formatTomlValue('10m')).toBe('"10m"')
    expect(formatTomlValue('2h')).toBe('"2h"')
    expect(formatTomlValue('1.5s')).toBe('"1.5s"')
  })

  it('handles JSON arrays and objects', () => {
    expect(formatTomlValue('["item1", "item2"]')).toBe(
      '"""["item1", "item2"]"""',
    )
    expect(formatTomlValue('{"key": "value"}')).toBe('"""{"key": "value"}"""')
  })

  it('handles invalid JSON as regular strings', () => {
    expect(formatTomlValue('[invalid json')).toBe('"[invalid json"')
    expect(formatTomlValue('{broken')).toBe('"{broken"')
  })

  it('handles regular strings with quote escaping', () => {
    expect(formatTomlValue('simple string')).toBe('"simple string"')
    expect(formatTomlValue('string with "quotes"')).toBe(
      '"string with \\"quotes\\""',
    )
    expect(formatTomlValue('/path/to/file')).toBe('"/path/to/file"')
  })

  it('handles strings with backslashes and quotes', () => {
    expect(formatTomlValue('C:\\Program Files\\app')).toBe(
      '"C:\\\\Program Files\\\\app"',
    )
    expect(formatTomlValue('path\\to"file')).toBe('"path\\\\to\\"file"')
    expect(formatTomlValue('backslash\\ and "quote"')).toBe(
      '"backslash\\\\ and \\"quote\\""',
    )
    expect(formatTomlValue('single\\backslash')).toBe('"single\\\\backslash"')
  })

  it('handles empty strings', () => {
    expect(formatTomlValue('')).toBe('""')
  })
})

describe('buildTomlSections', () => {
  it('separates root keys from nested keys', () => {
    const configs = [
      { key: 'RootKey1', value: 'value1' },
      { key: 'Section.NestedKey', value: 'value2' },
      { key: 'RootKey2', value: 'value3' },
      { key: 'Section.AnotherKey', value: 'value4' },
      { key: 'AnotherSection.Key', value: 'value5' },
    ]

    const result = buildTomlSections(configs)

    expect(result.rootKeys).toEqual([
      { key: 'RootKey1', value: 'value1' },
      { key: 'RootKey2', value: 'value3' },
    ])

    expect(result.sections).toEqual({
      Section: [
        { key: 'NestedKey', value: 'value2' },
        { key: 'AnotherKey', value: 'value4' },
      ],
      AnotherSection: [{ key: 'Key', value: 'value5' }],
    })
  })

  it('handles deeply nested keys', () => {
    const configs = [{ key: 'Section.SubSection.DeepKey', value: 'deepValue' }]

    const result = buildTomlSections(configs)

    expect(result.rootKeys).toEqual([])
    expect(result.sections).toEqual({
      Section: [{ key: 'SubSection.DeepKey', value: 'deepValue' }],
    })
  })

  it('handles empty input', () => {
    const result = buildTomlSections([])

    expect(result.rootKeys).toEqual([])
    expect(result.sections).toEqual({})
  })
})

describe('configToToml', () => {
  const mockTranslate = (key) => {
    const translations = {
      'about.config.devFlagsHeader':
        'Development Flags (subject to change/removal)',
      'about.config.devFlagsComment':
        'These are experimental settings and may be removed in future versions',
    }
    return translations[key] || key
  }

  it('generates TOML with header and timestamp', () => {
    const configData = {
      config: [{ key: 'TestKey', value: 'testValue' }],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('# Navidrome Configuration')
    expect(result).toContain('# Generated on')
    expect(result).toContain('TestKey = "testValue"')
  })

  it('separates and sorts regular and dev configs', () => {
    const configData = {
      config: [
        { key: 'ZRegularKey', value: 'regularValue' },
        { key: 'DevTestFlag', value: 'true' },
        { key: 'ARegularKey', value: 'anotherValue' },
        { key: 'DevAnotherFlag', value: 'false' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    // Check that regular configs come first and are sorted
    const lines = result.split('\n')
    const aRegularIndex = lines.findIndex((line) =>
      line.includes('ARegularKey'),
    )
    const zRegularIndex = lines.findIndex((line) =>
      line.includes('ZRegularKey'),
    )
    const devHeaderIndex = lines.findIndex((line) =>
      line.includes('Development Flags'),
    )
    const devAnotherIndex = lines.findIndex((line) =>
      line.includes('DevAnotherFlag'),
    )
    const devTestIndex = lines.findIndex((line) => line.includes('DevTestFlag'))

    expect(aRegularIndex).toBeLessThan(zRegularIndex)
    expect(zRegularIndex).toBeLessThan(devHeaderIndex)
    expect(devHeaderIndex).toBeLessThan(devAnotherIndex)
    expect(devAnotherIndex).toBeLessThan(devTestIndex)
  })

  it('skips ConfigFile entries', () => {
    const configData = {
      config: [
        { key: 'ConfigFile', value: '/path/to/config.toml' },
        { key: 'TestKey', value: 'testValue' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).not.toContain('ConfigFile =')
    expect(result).toContain('TestKey = "testValue"')
  })

  it('handles sections correctly', () => {
    const configData = {
      config: [
        { key: 'RootKey', value: 'rootValue' },
        { key: 'Section.NestedKey', value: 'nestedValue' },
        { key: 'Section.AnotherKey', value: 'anotherValue' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('RootKey = "rootValue"')
    expect(result).toContain('[Section]')
    expect(result).toContain('NestedKey = "nestedValue"')
    expect(result).toContain('AnotherKey = "anotherValue"')
  })

  it('includes dev flags header when dev configs exist', () => {
    const configData = {
      config: [
        { key: 'RegularKey', value: 'regularValue' },
        { key: 'DevTestFlag', value: 'true' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('# Development Flags (subject to change/removal)')
    expect(result).toContain(
      '# These are experimental settings and may be removed in future versions',
    )
    expect(result).toContain('DevTestFlag = true')
  })

  it('does not include dev flags header when no dev configs exist', () => {
    const configData = {
      config: [{ key: 'RegularKey', value: 'regularValue' }],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).not.toContain('Development Flags')
    expect(result).toContain('RegularKey = "regularValue"')
  })

  it('handles empty config data', () => {
    const configData = { config: [] }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('# Navidrome Configuration')
    expect(result).not.toContain('Development Flags')
  })

  it('handles missing config array', () => {
    const configData = {}

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('# Navidrome Configuration')
    expect(result).not.toContain('Development Flags')
  })

  it('works without translate function', () => {
    const configData = {
      config: [{ key: 'DevTestFlag', value: 'true' }],
    }

    const result = configToToml(configData)

    expect(result).toContain('# about.config.devFlagsHeader')
    expect(result).toContain('# about.config.devFlagsComment')
    expect(result).toContain('DevTestFlag = true')
  })

  it('handles various data types correctly', () => {
    const configData = {
      config: [
        { key: 'StringValue', value: 'test string' },
        { key: 'BooleanValue', value: 'true' },
        { key: 'IntegerValue', value: '42' },
        { key: 'FloatValue', value: '3.14' },
        { key: 'DurationValue', value: '5s' },
        { key: 'ArrayValue', value: '["item1", "item2"]' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('StringValue = "test string"')
    expect(result).toContain('BooleanValue = true')
    expect(result).toContain('IntegerValue = 42')
    expect(result).toContain('FloatValue = 3.14')
    expect(result).toContain('DurationValue = "5s"')
    expect(result).toContain('ArrayValue = """["item1", "item2"]"""')
  })
})

describe('separateAndSortConfigs', () => {
  it('separates regular and dev configs correctly', () => {
    const configs = [
      { key: 'RegularKey1', value: 'value1' },
      { key: 'DevTestFlag', value: 'true' },
      { key: 'AnotherRegular', value: 'value2' },
      { key: 'DevAnotherFlag', value: 'false' },
    ]

    const result = separateAndSortConfigs(configs)

    expect(result.regularConfigs).toEqual([
      { key: 'AnotherRegular', value: 'value2' },
      { key: 'RegularKey1', value: 'value1' },
    ])

    expect(result.devConfigs).toEqual([
      { key: 'DevAnotherFlag', value: 'false' },
      { key: 'DevTestFlag', value: 'true' },
    ])
  })

  it('skips ConfigFile entries', () => {
    const configs = [
      { key: 'ConfigFile', value: '/path/to/config.toml' },
      { key: 'RegularKey', value: 'value' },
      { key: 'DevFlag', value: 'true' },
    ]

    const result = separateAndSortConfigs(configs)

    expect(result.regularConfigs).toEqual([
      { key: 'RegularKey', value: 'value' },
    ])
    expect(result.devConfigs).toEqual([{ key: 'DevFlag', value: 'true' }])
  })

  it('handles empty input', () => {
    const result = separateAndSortConfigs([])

    expect(result.regularConfigs).toEqual([])
    expect(result.devConfigs).toEqual([])
  })

  it('handles null/undefined input', () => {
    const result1 = separateAndSortConfigs(null)
    const result2 = separateAndSortConfigs(undefined)

    expect(result1.regularConfigs).toEqual([])
    expect(result1.devConfigs).toEqual([])
    expect(result2.regularConfigs).toEqual([])
    expect(result2.devConfigs).toEqual([])
  })

  it('sorts configs alphabetically', () => {
    const configs = [
      { key: 'ZRegular', value: 'z' },
      { key: 'ARegular', value: 'a' },
      { key: 'DevZ', value: 'z' },
      { key: 'DevA', value: 'a' },
    ]

    const result = separateAndSortConfigs(configs)

    expect(result.regularConfigs[0].key).toBe('ARegular')
    expect(result.regularConfigs[1].key).toBe('ZRegular')
    expect(result.devConfigs[0].key).toBe('DevA')
    expect(result.devConfigs[1].key).toBe('DevZ')
  })
})
