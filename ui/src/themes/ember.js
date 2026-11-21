import { createVibrantTheme } from './vibrant'
import { ember } from './vibrantPalettes'

export const EmberDarkTheme = createVibrantTheme(
  'Ember Dark',
  'dark',
  ember.dark,
)
export const EmberLightTheme = createVibrantTheme(
  'Ember Light',
  'light',
  ember.light,
)
