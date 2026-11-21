import { createVibrantTheme } from './vibrant'
import { aurora } from './vibrantPalettes'

export const AuroraDarkTheme = createVibrantTheme(
  'Aurora Dark',
  'dark',
  aurora.dark,
)
export const AuroraLightTheme = createVibrantTheme(
  'Aurora Light',
  'light',
  aurora.light,
)
