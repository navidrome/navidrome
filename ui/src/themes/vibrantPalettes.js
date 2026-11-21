/**
 * Color palettes for the Vibrant theme family (see vibrant.js).
 *
 * Each palette has a dark and a light variant with these fields:
 *   background   page background
 *   paper        cards, tables, player bar
 *   elevated     dialogs, menus, inputs, chips
 *   divider      hairlines and borders
 *   text         main text
 *   textSecondary  subtitles and metadata
 *   accent       fills: buttons, play buttons, sliders, active indicators
 *   onAccent     text and icons on top of an accent fill
 *   accentText   accent used as text or icon color on the page background
 *   highlight    second accent (ratings, secondary color)
 *   onHighlight  text on top of a highlight fill
 *
 * vibrant.test.js checks the text pairs against WCAG AA contrast (4.5:1).
 */
export const ember = {
  name: 'Ember',
  dark: {
    background: '#121014',
    paper: '#1E1A1F',
    elevated: '#2A2429',
    divider: 'rgba(255, 255, 255, 0.08)',
    text: '#F5EFEC',
    textSecondary: '#B3A8A4',
    accent: '#FF6B4A',
    onAccent: '#1A0D09',
    accentText: '#FF6B4A',
    highlight: '#FFC24B',
    onHighlight: '#1A1300',
  },
  light: {
    background: '#FFF8F5',
    paper: '#FFFFFF',
    elevated: '#FCEDE6',
    divider: 'rgba(60, 30, 20, 0.1)',
    text: '#2B2220',
    textSecondary: '#75655F',
    accent: '#C2391F',
    onAccent: '#FFFFFF',
    accentText: '#C2391F',
    highlight: '#9A6100',
    onHighlight: '#FFFFFF',
  },
}

export const aurora = {
  name: 'Aurora',
  dark: {
    background: '#0E0B16',
    paper: '#18142A',
    elevated: '#221C3A',
    divider: 'rgba(255, 255, 255, 0.08)',
    text: '#F1EEFA',
    textSecondary: '#ABA3C2',
    accent: '#A78BFA',
    onAccent: '#120A24',
    accentText: '#A78BFA',
    highlight: '#2DD4BF',
    onHighlight: '#04201C',
  },
  light: {
    background: '#F7F5FF',
    paper: '#FFFFFF',
    elevated: '#EEEAFB',
    divider: 'rgba(40, 20, 90, 0.1)',
    text: '#1F1A2E',
    textSecondary: '#645C7A',
    accent: '#7C3AED',
    onAccent: '#FFFFFF',
    accentText: '#7C3AED',
    highlight: '#0F766E',
    onHighlight: '#FFFFFF',
  },
}

export const lagoon = {
  name: 'Lagoon',
  dark: {
    background: '#0B1214',
    paper: '#132024',
    elevated: '#1B2C31',
    divider: 'rgba(255, 255, 255, 0.08)',
    text: '#EAF4F3',
    textSecondary: '#9DB3B0',
    accent: '#14B8A6',
    onAccent: '#031A17',
    accentText: '#2DD4BF',
    highlight: '#A3E635',
    onHighlight: '#1A2600',
  },
  light: {
    background: '#F3FAF9',
    paper: '#FFFFFF',
    elevated: '#E3F2F0',
    divider: 'rgba(10, 60, 55, 0.1)',
    text: '#142321',
    textSecondary: '#57706C',
    accent: '#0F766E',
    onAccent: '#FFFFFF',
    accentText: '#0F766E',
    highlight: '#4D7C0F',
    onHighlight: '#FFFFFF',
  },
}

export const neonPop = {
  name: 'Neon Pop',
  dark: {
    background: '#0F0D12',
    paper: '#1A1620',
    elevated: '#25202D',
    divider: 'rgba(255, 255, 255, 0.08)',
    text: '#F7F0F5',
    textSecondary: '#B5A9B8',
    accent: '#FF3D8B',
    onAccent: '#1A0510',
    accentText: '#FF5C9E',
    highlight: '#5B8CFF',
    onHighlight: '#050B1F',
  },
  light: {
    background: '#FFF5F9',
    paper: '#FFFFFF',
    elevated: '#FCE4EE',
    divider: 'rgba(90, 10, 50, 0.1)',
    text: '#2A1520',
    textSecondary: '#735A66',
    accent: '#D6246E',
    onAccent: '#FFFFFF',
    accentText: '#C21E62',
    highlight: '#2457D6',
    onHighlight: '#FFFFFF',
  },
}

export default { ember, aurora, lagoon, neonPop }
