import { createVibrantTheme } from './vibrant'
import { neonPop } from './vibrantPalettes'

export const NeonPopDarkTheme = createVibrantTheme(
  'Neon Pop Dark',
  'dark',
  neonPop.dark,
)
export const NeonPopLightTheme = createVibrantTheme(
  'Neon Pop Light',
  'light',
  neonPop.light,
)
