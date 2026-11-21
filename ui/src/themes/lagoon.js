import { createVibrantTheme } from './vibrant'
import { lagoon } from './vibrantPalettes'

export const LagoonDarkTheme = createVibrantTheme(
  'Lagoon Dark',
  'dark',
  lagoon.dark,
)
export const LagoonLightTheme = createVibrantTheme(
  'Lagoon Light',
  'light',
  lagoon.light,
)
