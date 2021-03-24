import LightTheme from './light'
import DarkTheme from './dark'
import SpotifyTheme from './spotify'
import ExtraDark from './extradark'

export default { LightTheme, DarkTheme, SpotifyTheme, ExtraDark }

/**
 * @param {*} theme_colors
 * @param  {...array} this is rest of value which will be returned only if condition are true.
 * let the arguments provided as -
 * anyDomEle: {
 *  color: themeColorSelector('#696969',
 *                              [theme.palette.extraAttribute.theme, 'extradark', theme.palette.extraAttribute.subtitle],
 *                              [theme.palette.type, 'dark', '#c5c5c5'])
 * }
 * looks complicated but working is simple the function accept a default value, then array of array of 3 elements priority vise, this array elements are as follows [variable, is_variable_equals_to, this_variable]
 * working -
 * if(value[0][0] === value[0][1])
 *    return value[0][2]
 * else if(value[1][0] === value[1][1])
 *    return value[1][2]
 * else
 *    return defaultValue
 */
export function themeColorSelector(defaultValue, ...values) {
  console.log('DEF', defaultValue)
  for (let val in values) {
    if (values[val][0] === values[val][1]) {
      console.log('VALUE', values[val][0], values[val][1], values[val][2])
      return values[val][2]
    }
  }
  return defaultValue
}
