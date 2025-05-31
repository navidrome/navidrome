/**
 * TOML utility functions for configuration export
 */

/**
 * Flattens nested configuration object and generates environment variable names
 * @param {Object} config - The nested configuration object from the backend
 * @param {string} prefix - The current prefix for nested keys
 * @returns {Array} - Array of config objects with key, envVar, and value properties
 */
export const flattenConfig = (config, prefix = '') => {
  const result = []

  if (!config || typeof config !== 'object') {
    return result
  }

  Object.keys(config).forEach((key) => {
    const value = config[key]
    const currentKey = prefix ? `${prefix}.${key}` : key

    if (value && typeof value === 'object' && !Array.isArray(value)) {
      // Recursively flatten nested objects
      result.push(...flattenConfig(value, currentKey))
    } else {
      // Generate environment variable name: ND_ + uppercase with dots replaced by underscores
      const envVar = 'ND_' + currentKey.toUpperCase().replace(/\./g, '_')

      // Convert value to string for display
      let displayValue = value
      if (
        Array.isArray(value) ||
        (typeof value === 'object' && value !== null)
      ) {
        displayValue = JSON.stringify(value)
      } else {
        displayValue = String(value)
      }

      result.push({
        key: currentKey,
        envVar: envVar,
        value: displayValue,
      })
    }
  })

  return result
}

/**
 * Separates and sorts configuration entries into regular and dev configs
 * @param {Array|Object} configEntries - Array of config objects with key and value, or nested config object
 * @returns {Object} - Object with regularConfigs and devConfigs arrays, both sorted
 */
export const separateAndSortConfigs = (configEntries) => {
  const regularConfigs = []
  const devConfigs = []

  // Handle both the old array format and new nested object format
  let flattenedConfigs
  if (Array.isArray(configEntries)) {
    // Old format - already flattened
    flattenedConfigs = configEntries
  } else {
    // New format - need to flatten
    flattenedConfigs = flattenConfig(configEntries)
  }

  flattenedConfigs?.forEach((config) => {
    // Skip configFile as it's displayed separately
    if (config.key === 'ConfigFile') {
      return
    }

    if (config.key.startsWith('Dev')) {
      devConfigs.push(config)
    } else {
      regularConfigs.push(config)
    }
  })

  // Sort configurations alphabetically
  regularConfigs.sort((a, b) => a.key.localeCompare(b.key))
  devConfigs.sort((a, b) => a.key.localeCompare(b.key))

  return { regularConfigs, devConfigs }
}

/**
 * Escapes TOML keys that contain special characters
 * @param {string} key - The key to potentially escape
 * @returns {string} - The escaped key if needed, or the original key
 */
export const escapeTomlKey = (key) => {
  // Convert to string first to handle null/undefined
  const keyStr = String(key)

  // Empty strings always need quotes
  if (keyStr === '') {
    return '""'
  }

  // TOML bare keys can only contain letters, numbers, underscores, and hyphens
  // If the key contains other characters, it needs to be quoted
  if (/^[a-zA-Z0-9_-]+$/.test(keyStr)) {
    return keyStr
  }

  // Escape quotes in the key and wrap in quotes
  return `"${keyStr.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

/**
 * Converts a value to proper TOML format
 * @param {*} value - The value to format
 * @returns {string} - The TOML-formatted value
 */
export const formatTomlValue = (value) => {
  if (value === null || value === undefined) {
    return '""'
  }

  const str = String(value)

  // Boolean values
  if (str === 'true' || str === 'false') {
    return str
  }

  // Numbers (integers and floats)
  if (/^-?\d+$/.test(str)) {
    return str // Integer
  }
  if (/^-?\d*\.\d+$/.test(str)) {
    return str // Float
  }

  // Duration values (like "300ms", "1s", "5m")
  if (/^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)$/.test(str)) {
    return `"${str}"`
  }

  // Handle arrays and objects
  if (str.startsWith('[') || str.startsWith('{')) {
    try {
      const parsed = JSON.parse(str)

      // If it's an array, format as TOML array
      if (Array.isArray(parsed)) {
        const formattedItems = parsed.map((item) => {
          if (typeof item === 'string') {
            return `"${item.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
          } else if (typeof item === 'number' || typeof item === 'boolean') {
            return String(item)
          } else {
            return `"${String(item).replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
          }
        })

        if (formattedItems.length === 0) {
          return '[ ]'
        }
        return `[ ${formattedItems.join(', ')} ]`
      }

      // For objects, keep the JSON string format with triple quotes
      return `"""${str}"""`
    } catch {
      return `"${str.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
    }
  }

  // String values (escape backslashes and quotes)
  return `"${str.replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`
}

/**
 * Converts nested keys to TOML sections
 * @param {Array} configs - Array of config objects with key and value
 * @returns {Object} - Object with sections and rootKeys
 */
export const buildTomlSections = (configs) => {
  const sections = {}
  const rootKeys = []

  configs.forEach(({ key, value }) => {
    if (key.includes('.')) {
      const parts = key.split('.')
      const sectionName = parts[0]
      const keyName = parts.slice(1).join('.')

      if (!sections[sectionName]) {
        sections[sectionName] = []
      }
      sections[sectionName].push({ key: keyName, value })
    } else {
      rootKeys.push({ key, value })
    }
  })

  return { sections, rootKeys }
}

/**
 * Converts configuration data to TOML format
 * @param {Object} configData - The configuration data object
 * @param {Function} translate - Translation function for internationalization
 * @returns {string} - The TOML-formatted configuration
 */
export const configToToml = (configData, translate = (key) => key) => {
  let tomlContent = `# Navidrome Configuration\n# Generated on ${new Date().toISOString()}\n\n`

  // Handle both old array format (configData.config is array) and new nested format (configData.config is object)
  let configs
  if (Array.isArray(configData.config)) {
    // Old format - already flattened
    configs = configData.config
  } else {
    // New format - need to flatten
    configs = flattenConfig(configData.config)
  }

  const { regularConfigs, devConfigs } = separateAndSortConfigs(configs)

  // Process regular configs
  const { sections: regularSections, rootKeys: regularRootKeys } =
    buildTomlSections(regularConfigs)

  // Add root-level keys first
  if (regularRootKeys.length > 0) {
    regularRootKeys.forEach(({ key, value }) => {
      tomlContent += `${key} = ${formatTomlValue(value)}\n`
    })
    tomlContent += '\n'
  }

  // Add dev configs if any
  if (devConfigs.length > 0) {
    tomlContent += `# ${translate('about.config.devFlagsHeader')}\n`
    tomlContent += `# ${translate('about.config.devFlagsComment')}\n\n`

    const { sections: devSections, rootKeys: devRootKeys } =
      buildTomlSections(devConfigs)

    // Add dev root-level keys
    devRootKeys.forEach(({ key, value }) => {
      tomlContent += `${key} = ${formatTomlValue(value)}\n`
    })
    if (devRootKeys.length > 0) {
      tomlContent += '\n'
    }

    // Add dev sections
    Object.keys(devSections)
      .sort()
      .forEach((sectionName) => {
        tomlContent += `[${sectionName}]\n`
        devSections[sectionName].forEach(({ key, value }) => {
          tomlContent += `${escapeTomlKey(key)} = ${formatTomlValue(value)}\n`
        })
        tomlContent += '\n'
      })
  }

  // Add sections
  Object.keys(regularSections)
    .sort()
    .forEach((sectionName) => {
      tomlContent += `[${sectionName}]\n`
      regularSections[sectionName].forEach(({ key, value }) => {
        tomlContent += `${escapeTomlKey(key)} = ${formatTomlValue(value)}\n`
      })
      tomlContent += '\n'
    })

  return tomlContent
}
