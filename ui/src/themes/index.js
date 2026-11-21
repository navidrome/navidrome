import LightTheme from './light'
import DarkTheme from './dark'
import ExtraDarkTheme from './extradark'
import GreenTheme from './green'
import SpotifyTheme from './spotify'
import { LagoonDarkTheme, LagoonLightTheme } from './lagoon'
import LigeraTheme from './ligera'
import MonokaiTheme from './monokai'
import ElectricPurpleTheme from './electricPurple'
import NordTheme from './nord'
import GruvboxDarkTheme from './gruvboxDark'
import CatppuccinFrappeTheme from './catppuccinFrappe'
import CatppuccinMacchiatoTheme from './catppuccinMacchiato'
import CatppuccinMochaTheme from './catppuccinMocha'
import CatppuccinLatteTheme from './catppuccinLatte'
import DraculaTheme from './dracula'
import { EmberDarkTheme, EmberLightTheme } from './ember'
import NuclearTheme from './nuclear'
import NutballTheme from './nutball'
import RosePineTheme from './rosePine'
import RosePineDawnTheme from './rosePineDawn'
import RosePineMoonTheme from './rosePineMoon'
import AmusicTheme from './amusic'
import { AuroraDarkTheme, AuroraLightTheme } from './aurora'
import CaveTheme from './cave'
import SquiddiesGlassTheme from './SquiddiesGlass'
import NautilineTheme from './nautiline'
import { NeonPopDarkTheme, NeonPopLightTheme } from './neonPop'
import MoonbaseAlphaTheme from './moonbaseAlpha'
import MoonbaseBravoTheme from './moonbaseBravo'
import TokyoNightLightTheme from './tokyoNightLight'
import TokyoNightTheme from './tokyoNight'
import SolarizedDarkTheme from './solarizedDark'
import SolarizedLightTheme from './solarizedLight'
import vibrantPalettes from './vibrantPalettes'

const themes = {
  // Classic default themes
  LightTheme,
  DarkTheme,

  // New themes should be added here, in alphabetic order
  AmusicTheme,
  AuroraDarkTheme,
  AuroraLightTheme,
  CatppuccinFrappeTheme,
  CatppuccinLatteTheme,
  CatppuccinMacchiatoTheme,
  CatppuccinMochaTheme,
  CaveTheme,
  DraculaTheme,
  ElectricPurpleTheme,
  EmberDarkTheme,
  EmberLightTheme,
  ExtraDarkTheme,
  GreenTheme,
  GruvboxDarkTheme,
  LagoonDarkTheme,
  LagoonLightTheme,
  LigeraTheme,
  MonokaiTheme,
  MoonbaseAlphaTheme,
  MoonbaseBravoTheme,
  NautilineTheme,
  NeonPopDarkTheme,
  NeonPopLightTheme,
  NordTheme,
  NuclearTheme,
  NutballTheme,
  RosePineDawnTheme,
  RosePineMoonTheme,
  RosePineTheme,
  SolarizedDarkTheme,
  SolarizedLightTheme,
  SpotifyTheme,
  SquiddiesGlassTheme,
  TokyoNightLightTheme,
  TokyoNightTheme,
}

export default themes

// TEMPORARY (local fork): while the Vibrant family is being developed, the theme picker only
// offers Vibrant themes (one Dark and one Light per palette in vibrantPalettes.js). All other
// themes stay registered above, so saved preferences, the DefaultTheme setting and tests still
// resolve them. To show every theme again, make this return true.
const pickerThemeNames = Object.values(vibrantPalettes).flatMap((p) => [
  `${p.name} Dark`,
  `${p.name} Light`,
])
export const isThemeInPicker = (key) =>
  pickerThemeNames.includes(themes[key]?.themeName)
