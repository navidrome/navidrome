import { describe, it, expect } from 'vitest'
import {
  formatTomlValue,
  buildTomlSections,
  configToToml,
  separateAndSortConfigs,
  flattenConfig,
  escapeTomlKey,
} from './aboutUtils'

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
    expect(formatTomlValue('["item1", "item2"]')).toBe('[ "item1", "item2" ]')
    expect(formatTomlValue('{"key": "value"}')).toBe('"""{"key": "value"}"""')
  })

  it('formats different types of arrays correctly', () => {
    // String array
    expect(formatTomlValue('["genre", "tcon", "©gen"]')).toBe(
      '[ "genre", "tcon", "©gen" ]',
    )
    // Mixed array with numbers and strings
    expect(formatTomlValue('[42, "test", true]')).toBe('[ 42, "test", true ]')
    // Empty array
    expect(formatTomlValue('[]')).toBe('[ ]')
    // Array with special characters in strings
    expect(
      formatTomlValue('["item with spaces", "item\\"with\\"quotes"]'),
    ).toBe('[ "item with spaces", "item\\"with\\"quotes" ]')
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
        { key: 'DevA', value: 'DevValue' },
      ],
    }

    const result = configToToml(configData, mockTranslate)
    // Fields in a section are sorted alphabetically
    const fields = [
      'RootKey = "rootValue"',
      'DevA = "DevValue"',
      '[Section]',
      'AnotherKey = "anotherValue"',
      'NestedKey = "nestedValue"',
    ]

    for (let idx = 0; idx < fields.length - 1; idx++) {
      expect(result).toContain(fields[idx])

      const idxA = result.indexOf(fields[idx])
      const idxB = result.indexOf(fields[idx + 1])

      expect(idxA).toBeLessThan(idxB)
    }

    expect(result).toContain(fields[fields.length - 1])
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
    expect(result).toContain('ArrayValue = [ "item1", "item2" ]')
  })

  it('handles nested config object format correctly', () => {
    const configData = {
      config: {
        Address: '127.0.0.1',
        Port: 4533,
        EnableDownloads: true,
        DevLogSourceLine: false,
        LastFM: {
          Enabled: true,
          ApiKey: 'secret123',
          Language: 'en',
        },
        Scanner: {
          Schedule: 'daily',
          Enabled: true,
        },
      },
    }

    const result = configToToml(configData, mockTranslate)

    // Should contain regular configs
    expect(result).toContain('Address = "127.0.0.1"')
    expect(result).toContain('Port = 4533')
    expect(result).toContain('EnableDownloads = true')

    // Should contain dev configs with header
    expect(result).toContain('# Development Flags (subject to change/removal)')
    expect(result).toContain('DevLogSourceLine = false')

    // Should contain sections
    expect(result).toContain('[LastFM]')
    expect(result).toContain('Enabled = true')
    expect(result).toContain('ApiKey = "secret123"')
    expect(result).toContain('Language = "en"')

    expect(result).toContain('[Scanner]')
    expect(result).toContain('Schedule = "daily"')
  })

  it('handles mixed nested and flat structure', () => {
    const configData = {
      config: {
        MusicFolder: '/music',
        DevAutoLoginUsername: 'testuser',
        Jukebox: {
          Enabled: false,
          AdminOnly: true,
        },
      },
    }

    const result = configToToml(configData, mockTranslate)

    expect(result).toContain('MusicFolder = "/music"')
    expect(result).toContain('DevAutoLoginUsername = "testuser"')
    expect(result).toContain('[Jukebox]')
    expect(result).toContain('Enabled = false')
    expect(result).toContain('AdminOnly = true')
  })

  it('properly escapes keys with special characters in sections', () => {
    const configData = {
      config: [
        { key: 'DevLogLevels.persistence/sql_base_repository', value: 'trace' },
        { key: 'DevLogLevels.core/scanner', value: 'debug' },
        { key: 'DevLogLevels.regular_key', value: 'info' },
        { key: 'Tags.genre.Aliases', value: '["tcon","genre","©gen"]' },
      ],
    }

    const result = configToToml(configData, mockTranslate)

    // Keys with forward slashes should be quoted
    expect(result).toContain('"persistence/sql_base_repository" = "trace"')
    expect(result).toContain('"core/scanner" = "debug"')

    // Regular keys should not be quoted
    expect(result).toContain('regular_key = "info"')

    // Arrays should be formatted correctly
    expect(result).toContain('"genre.Aliases" = [ "tcon", "genre", "©gen" ]')

    // Should contain proper sections
    expect(result).toContain('[DevLogLevels]')
    expect(result).toContain('[Tags]')
  })
})

describe('flattenConfig', () => {
  it('flattens simple nested objects correctly', () => {
    const config = {
      Address: '0.0.0.0',
      Port: 4533,
      EnableDownloads: true,
      LastFM: {
        Enabled: true,
        ApiKey: 'secret123',
        Language: 'en',
      },
    }

    const result = flattenConfig(config)

    expect(result).toContainEqual({
      key: 'Address',
      envVar: 'ND_ADDRESS',
      value: '0.0.0.0',
    })

    expect(result).toContainEqual({
      key: 'Port',
      envVar: 'ND_PORT',
      value: '4533',
    })

    expect(result).toContainEqual({
      key: 'EnableDownloads',
      envVar: 'ND_ENABLEDOWNLOADS',
      value: 'true',
    })

    expect(result).toContainEqual({
      key: 'LastFM.Enabled',
      envVar: 'ND_LASTFM_ENABLED',
      value: 'true',
    })

    expect(result).toContainEqual({
      key: 'LastFM.ApiKey',
      envVar: 'ND_LASTFM_APIKEY',
      value: 'secret123',
    })

    expect(result).toContainEqual({
      key: 'LastFM.Language',
      envVar: 'ND_LASTFM_LANGUAGE',
      value: 'en',
    })
  })

  it('handles deeply nested objects', () => {
    const config = {
      Scanner: {
        Schedule: 'daily',
        Options: {
          ExtractorType: 'taglib',
          ArtworkPriority: 'cover.jpg',
        },
      },
    }

    const result = flattenConfig(config)

    expect(result).toContainEqual({
      key: 'Scanner.Schedule',
      envVar: 'ND_SCANNER_SCHEDULE',
      value: 'daily',
    })

    expect(result).toContainEqual({
      key: 'Scanner.Options.ExtractorType',
      envVar: 'ND_SCANNER_OPTIONS_EXTRACTORTYPE',
      value: 'taglib',
    })

    expect(result).toContainEqual({
      key: 'Scanner.Options.ArtworkPriority',
      envVar: 'ND_SCANNER_OPTIONS_ARTWORKPRIORITY',
      value: 'cover.jpg',
    })
  })

  it('handles arrays correctly', () => {
    const config = {
      DeviceList: ['device1', 'device2'],
      Settings: {
        EnabledFormats: ['mp3', 'flac', 'ogg'],
      },
    }

    const result = flattenConfig(config)

    expect(result).toContainEqual({
      key: 'DeviceList',
      envVar: 'ND_DEVICELIST',
      value: '["device1","device2"]',
    })

    expect(result).toContainEqual({
      key: 'Settings.EnabledFormats',
      envVar: 'ND_SETTINGS_ENABLEDFORMATS',
      value: '["mp3","flac","ogg"]',
    })
  })

  it('handles null and undefined values', () => {
    const config = {
      NullValue: null,
      UndefinedValue: undefined,
      EmptyString: '',
      ZeroValue: 0,
    }

    const result = flattenConfig(config)

    expect(result).toContainEqual({
      key: 'NullValue',
      envVar: 'ND_NULLVALUE',
      value: 'null',
    })

    expect(result).toContainEqual({
      key: 'UndefinedValue',
      envVar: 'ND_UNDEFINEDVALUE',
      value: 'undefined',
    })

    expect(result).toContainEqual({
      key: 'EmptyString',
      envVar: 'ND_EMPTYSTRING',
      value: '',
    })

    expect(result).toContainEqual({
      key: 'ZeroValue',
      envVar: 'ND_ZEROVALUE',
      value: '0',
    })
  })

  it('handles empty object', () => {
    const result = flattenConfig({})
    expect(result).toEqual([])
  })

  it('handles null/undefined input', () => {
    expect(flattenConfig(null)).toEqual([])
    expect(flattenConfig(undefined)).toEqual([])
  })

  it('handles non-object input', () => {
    expect(flattenConfig('string')).toEqual([])
    expect(flattenConfig(123)).toEqual([])
    expect(flattenConfig(true)).toEqual([])
  })
})

describe('separateAndSortConfigs', () => {
  it('separates regular and dev configs correctly with array input', () => {
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

  it('separates regular and dev configs correctly with nested object input', () => {
    const config = {
      Address: '127.0.0.1',
      Port: 4533,
      DevAutoLoginUsername: 'testuser',
      DevLogSourceLine: true,
      LastFM: {
        Enabled: true,
        ApiKey: 'secret123',
      },
    }

    const result = separateAndSortConfigs(config)

    expect(result.regularConfigs).toEqual([
      { key: 'Address', envVar: 'ND_ADDRESS', value: '127.0.0.1' },
      { key: 'LastFM.ApiKey', envVar: 'ND_LASTFM_APIKEY', value: 'secret123' },
      { key: 'LastFM.Enabled', envVar: 'ND_LASTFM_ENABLED', value: 'true' },
      { key: 'Port', envVar: 'ND_PORT', value: '4533' },
    ])

    expect(result.devConfigs).toEqual([
      {
        key: 'DevAutoLoginUsername',
        envVar: 'ND_DEVAUTOLOGINUSERNAME',
        value: 'testuser',
      },
      { key: 'DevLogSourceLine', envVar: 'ND_DEVLOGSOURCELINE', value: 'true' },
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

  it('skips ConfigFile entries with nested object input', () => {
    const config = {
      ConfigFile: '/path/to/config.toml',
      RegularKey: 'value',
      DevFlag: true,
    }

    const result = separateAndSortConfigs(config)

    expect(result.regularConfigs).toEqual([
      { key: 'RegularKey', envVar: 'ND_REGULARKEY', value: 'value' },
    ])
    expect(result.devConfigs).toEqual([
      { key: 'DevFlag', envVar: 'ND_DEVFLAG', value: 'true' },
    ])
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

describe('escapeTomlKey', () => {
  it('does not escape valid bare keys', () => {
    expect(escapeTomlKey('RegularKey')).toBe('RegularKey')
    expect(escapeTomlKey('regular_key')).toBe('regular_key')
    expect(escapeTomlKey('regular-key')).toBe('regular-key')
    expect(escapeTomlKey('key123')).toBe('key123')
    expect(escapeTomlKey('Key_with_underscores')).toBe('Key_with_underscores')
    expect(escapeTomlKey('Key-with-hyphens')).toBe('Key-with-hyphens')
  })

  it('escapes keys with special characters', () => {
    // Keys with forward slashes (like DevLogLevels keys)
    expect(escapeTomlKey('persistence/sql_base_repository')).toBe(
      '"persistence/sql_base_repository"',
    )
    expect(escapeTomlKey('core/scanner')).toBe('"core/scanner"')

    // Keys with dots
    expect(escapeTomlKey('Section.NestedKey')).toBe('"Section.NestedKey"')

    // Keys with spaces
    expect(escapeTomlKey('key with spaces')).toBe('"key with spaces"')

    // Keys with other special characters
    expect(escapeTomlKey('key@with@symbols')).toBe('"key@with@symbols"')
    expect(escapeTomlKey('key+with+plus')).toBe('"key+with+plus"')
  })

  it('escapes quotes in keys', () => {
    expect(escapeTomlKey('key"with"quotes')).toBe('"key\\"with\\"quotes"')
    expect(escapeTomlKey('key with "quotes" inside')).toBe(
      '"key with \\"quotes\\" inside"',
    )
  })

  it('escapes backslashes in keys', () => {
    expect(escapeTomlKey('key\\with\\backslashes')).toBe(
      '"key\\\\with\\\\backslashes"',
    )
    expect(escapeTomlKey('path\\to\\file')).toBe('"path\\\\to\\\\file"')
  })

  it('handles empty and null keys', () => {
    expect(escapeTomlKey('')).toBe('""')
    expect(escapeTomlKey(null)).toBe('null')
    expect(escapeTomlKey(undefined)).toBe('undefined')
  })
})
