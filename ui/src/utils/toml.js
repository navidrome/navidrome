/**
 * TOML utility functions for configuration export
 */

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
  
  // Arrays/JSON objects
  if (str.startsWith('[') || str.startsWith('{')) {
    try {
      JSON.parse(str)
      return `"""${str}"""`
    } catch {
      return `"${str.replace(/"/g, '\\"')}"`
    }
  }
  
  // String values (escape quotes)
  return `"${str.replace(/"/g, '\\"')}"`
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
 * @returns {string} - The TOML-formatted configuration
 */
export const configToToml = (configData) => {
  let tomlContent = `# Navidrome Configuration\n# Generated on ${new Date().toISOString()}\n\n`
  
  // Separate regular and dev configs
  const regularConfigs = []
  const devConfigs = []
  
  configData.config?.forEach(({ key, value }) => {
    // Skip configFile as it's displayed separately
    if (key === 'ConfigFile') {
      return
    }
    
    if (key.startsWith('Dev')) {
      devConfigs.push({ key, value })
    } else {
      regularConfigs.push({ key, value })
    }
  })

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
  
  // Add sections
  Object.keys(regularSections)
    .sort()
    .forEach((sectionName) => {
      tomlContent += `[${sectionName}]\n`
      regularSections[sectionName].forEach(({ key, value }) => {
        tomlContent += `${key} = ${formatTomlValue(value)}\n`
      })
      tomlContent += '\n'
    })

  // Add dev configs if any
  if (devConfigs.length > 0) {
    tomlContent += `# Development Flags (subject to change/removal)\n`
    tomlContent += `# These are experimental settings and may be removed in future versions\n\n`
    
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
          tomlContent += `${key} = ${formatTomlValue(value)}\n`
        })
        tomlContent += '\n'
      })
  }

  return tomlContent
} 