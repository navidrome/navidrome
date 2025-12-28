/**
 * Validates a JSON string and returns validation result
 * @param {string} value - The JSON string to validate
 * @returns {{ valid: boolean, error: string|null, parsed: object|null }}
 */
export const validateJson = (value) => {
  if (!value || value.trim() === '') {
    return { valid: true, error: null, parsed: null }
  }

  try {
    const parsed = JSON.parse(value)
    // Ensure config is an object, not an array or primitive
    if (
      typeof parsed !== 'object' ||
      parsed === null ||
      Array.isArray(parsed)
    ) {
      return {
        valid: false,
        error: 'Configuration must be a JSON object',
        parsed: null,
      }
    }
    return { valid: true, error: null, parsed }
  } catch (e) {
    // Try to provide helpful error messages
    let error = 'Invalid JSON'

    if (e instanceof SyntaxError) {
      const message = e.message

      // Extract position information if available
      const positionMatch = message.match(/position (\d+)/)
      if (positionMatch) {
        const position = parseInt(positionMatch[1], 10)
        const lines = value.substring(0, position).split('\n')
        const line = lines.length
        const column = lines[lines.length - 1].length + 1
        error = `Invalid JSON at line ${line}, column ${column}`
      } else if (message.includes('Unexpected end of JSON')) {
        error = 'Incomplete JSON - check for missing brackets or quotes'
      } else if (message.includes('Unexpected token')) {
        error = 'Invalid JSON - unexpected character found'
      }
    }

    return { valid: false, error, parsed: null }
  }
}

/**
 * Formats JSON string with proper indentation
 * @param {string} value - The JSON string to format
 * @returns {string} - Formatted JSON string or original if invalid
 */
export const formatJson = (value) => {
  if (!value || value.trim() === '') {
    return value
  }

  try {
    const parsed = JSON.parse(value)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return value
  }
}
