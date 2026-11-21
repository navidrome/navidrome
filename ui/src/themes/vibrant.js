/**
 * Vibrant theme family: bold accents, pill-shaped controls and large rounded corners.
 *
 * Every palette in vibrantPalettes.js is built in a dark and a light variant from this one
 * factory, so shape, typography and component styling stay identical across the family and
 * only the colors change.
 */
import createVibrantPlayerStylesheet from './vibrant.css.js'

export const radii = {
  card: '24px', // cards, dialogs, login box
  art: '16px', // album, playlist and artist artwork
  control: '12px', // inputs, menus, snackbars
  small: '8px', // menu items, tooltips
  pill: '999px', // buttons, chips, tabs
  round: '50%',
}

export const hexToRgba = (hex, alpha) => {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  if (!m) return hex
  const [r, g, b] = m.slice(1).map((v) => parseInt(v, 16))
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

const fontFamily = [
  'Inter',
  '-apple-system',
  'BlinkMacSystemFont',
  '"Segoe UI"',
  'Roboto',
  '"Helvetica Neue"',
  'Arial',
  'sans-serif',
].join(',')

const heading = (fontWeight, letterSpacing = '-0.02em') => ({
  fontWeight,
  letterSpacing,
})

// The album and playlist action bars put Play first: render it as a filled accent pill.
const actionBarStyle = (c) => ({
  '& button:first-of-type': {
    backgroundColor: c.accent,
    color: c.onAccent,
    padding: '6px 20px',
    marginRight: '8px',
    '& svg': { color: c.onAccent },
    '&:hover': {
      backgroundColor: c.accent,
      filter: 'brightness(1.08)',
      boxShadow: `0 6px 18px ${c.accentGlow}`,
    },
  },
})

const activeMenuLink = (c) => ({
  color: `${c.accentText} !important`,
  backgroundColor: c.accentSoft,
  '& .MuiListItemIcon-root': {
    color: `${c.accentText} !important`,
  },
})

/**
 * Builds one theme from a palette mode definition (see vibrantPalettes.js for the fields).
 */
export const createVibrantTheme = (themeName, mode, palette) => {
  const isDark = mode === 'dark'
  const c = {
    ...palette,
    accentSoft: hexToRgba(palette.accent, isDark ? 0.16 : 0.1),
    accentHover: hexToRgba(palette.accent, isDark ? 0.1 : 0.06),
    accentGlow: hexToRgba(palette.accent, isDark ? 0.35 : 0.3),
    shadow: isDark ? 'rgba(0, 0, 0, 0.45)' : 'rgba(60, 30, 20, 0.08)',
  }
  const surfaceShadow = isDark
    ? `0 0 0 1px ${c.divider}, 0 8px 24px ${c.shadow}`
    : `0 1px 3px ${c.shadow}, 0 8px 24px ${c.shadow}`

  return {
    themeName,
    palette: {
      type: mode,
      primary: { main: c.accent, contrastText: c.onAccent },
      secondary: { main: c.highlight, contrastText: c.onHighlight },
      background: { default: c.background, paper: c.paper },
      text: { primary: c.text, secondary: c.textSecondary },
      divider: c.divider,
      action: { hover: c.accentHover, selected: c.accentSoft },
    },
    shape: { borderRadius: 12 },
    typography: {
      fontFamily,
      h1: heading(800, '-0.03em'),
      h2: heading(800, '-0.03em'),
      h3: heading(800),
      h4: heading(800),
      h5: heading(700),
      h6: heading(700, '-0.01em'),
      button: { fontWeight: 700, textTransform: 'none' },
    },
    overrides: {
      MuiCssBaseline: {
        '@global': {
          body: { backgroundColor: c.background },
          '*::-webkit-scrollbar-thumb': {
            backgroundColor: c.elevated,
            borderRadius: radii.pill,
          },
        },
      },
      MuiAppBar: {
        root: { boxShadow: 'none', borderBottom: `1px solid ${c.divider}` },
        colorSecondary: { backgroundColor: c.background, color: c.text },
      },
      MuiPaper: {
        root: { backgroundColor: c.paper, color: c.text },
        rounded: { borderRadius: radii.card },
        elevation1: { boxShadow: surfaceShadow },
        elevation2: { boxShadow: surfaceShadow },
      },
      MuiCard: {
        root: { borderRadius: radii.card, overflow: 'hidden' },
      },
      MuiDialog: {
        paper: { borderRadius: radii.card, backgroundColor: c.elevated },
      },
      MuiPopover: {
        paper: { borderRadius: radii.control },
      },
      MuiMenu: {
        paper: { borderRadius: radii.control, backgroundColor: c.elevated },
      },
      MuiMenuItem: {
        root: { borderRadius: radii.small, margin: '2px 6px' },
      },
      MuiButton: {
        root: { borderRadius: radii.pill, padding: '6px 16px' },
        contained: { boxShadow: 'none', '&:hover': { boxShadow: 'none' } },
        containedPrimary: {
          backgroundColor: c.accent,
          color: c.onAccent,
          '&:hover': {
            backgroundColor: c.accent,
            filter: 'brightness(1.08)',
            boxShadow: `0 6px 18px ${c.accentGlow}`,
          },
        },
        outlinedPrimary: { borderColor: c.accent, color: c.accentText },
        textPrimary: { color: c.accentText },
      },
      MuiIconButton: {
        root: { '&:hover': { backgroundColor: c.accentHover } },
        colorPrimary: { color: c.accentText },
      },
      MuiFab: {
        root: { boxShadow: `0 6px 18px ${c.accentGlow}` },
        primary: {
          backgroundColor: c.accent,
          color: c.onAccent,
          '&:hover': { backgroundColor: c.accent, filter: 'brightness(1.08)' },
        },
      },
      MuiChip: {
        root: {
          borderRadius: radii.pill,
          backgroundColor: c.elevated,
          color: c.text,
          fontWeight: 600,
        },
        clickable: {
          '&:hover, &:focus': { backgroundColor: c.accentSoft },
        },
        colorPrimary: { backgroundColor: c.accent, color: c.onAccent },
      },
      MuiTabs: {
        indicator: {
          height: 3,
          borderRadius: radii.pill,
          backgroundColor: c.accent,
        },
      },
      MuiTab: {
        root: {
          textTransform: 'none',
          fontWeight: 700,
          '&$selected': { color: c.accentText },
        },
      },
      MuiTableCell: {
        root: { borderBottom: `1px solid ${c.divider}` },
        head: {
          backgroundColor: c.paper,
          color: c.textSecondary,
          fontWeight: 700,
          fontSize: '0.75rem',
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
        },
      },
      MuiTableRow: {
        hover: {
          '&:hover': { backgroundColor: `${c.accentHover} !important` },
        },
      },
      MuiListItem: {
        button: { '&:hover': { backgroundColor: c.accentHover } },
        root: {
          '&$selected, &$selected:hover': {
            backgroundColor: c.accentSoft,
            color: c.accentText,
          },
        },
      },
      MuiDrawer: {
        paper: { backgroundColor: c.background, borderRight: 'none' },
      },
      RaMenuItemLink: {
        root: {
          borderRadius: `0 ${radii.pill} ${radii.pill} 0`,
          marginRight: 8,
          '&[class*="makeStyles-active"]': activeMenuLink(c),
        },
        active: activeMenuLink(c),
      },
      MuiFilledInput: {
        root: {
          borderRadius: radii.control,
          backgroundColor: c.elevated,
          '&:hover': { backgroundColor: c.elevated },
          '&$focused': {
            backgroundColor: c.elevated,
            boxShadow: `0 0 0 2px ${c.accent}`,
          },
        },
        underline: {
          '&:before, &:after, &:hover:not($disabled):before': {
            borderBottom: 'none',
          },
        },
      },
      MuiOutlinedInput: {
        root: {
          borderRadius: radii.control,
          '&$focused $notchedOutline': { borderColor: c.accent },
        },
      },
      MuiCheckbox: {
        colorPrimary: { '&$checked': { color: c.accent } },
        colorSecondary: { '&$checked': { color: c.accent } },
      },
      MuiRadio: {
        colorPrimary: { '&$checked': { color: c.accent } },
        colorSecondary: { '&$checked': { color: c.accent } },
      },
      MuiSlider: {
        root: { color: c.accent },
        rail: { backgroundColor: c.textSecondary },
      },
      MuiLinearProgress: {
        root: { borderRadius: radii.pill, backgroundColor: c.elevated },
        bar: { borderRadius: radii.pill, backgroundColor: c.accent },
      },
      MuiRating: {
        iconFilled: { color: c.highlight },
        iconHover: { color: c.highlight },
      },
      MuiTooltip: {
        tooltip: {
          backgroundColor: c.text,
          color: c.background,
          borderRadius: radii.small,
          fontSize: '0.75rem',
          fontWeight: 600,
        },
      },
      MuiSnackbarContent: {
        root: { borderRadius: radii.control },
      },
      MuiBottomNavigation: {
        root: { backgroundColor: c.paper, borderTop: `1px solid ${c.divider}` },
      },
      MuiBottomNavigationAction: {
        root: { '&$selected': { color: c.accentText } },
      },
      NDLogin: {
        card: {
          borderRadius: radii.card,
          backgroundColor: hexToRgba(c.paper, 0.92),
          boxShadow: `0 24px 64px ${c.shadow}`,
        },
        systemNameLink: { color: c.accentText },
        welcome: { color: c.text },
        button: { boxShadow: `0 6px 18px ${c.accentGlow}` },
      },
      NDAlbumGridView: {
        // Round the link, not the image: the hover overlay is a sibling of the image
        // and must be clipped with it (enforced by theme.test.js).
        link: { borderRadius: radii.art, overflow: 'hidden' },
        albumName: { color: c.text, fontWeight: 700 },
        albumSubtitle: { color: c.textSecondary },
        albumVersion: { color: c.textSecondary },
        albumPlayButton: {
          backgroundColor: c.accent,
          color: c.onAccent,
          borderRadius: radii.round,
          padding: 8,
          boxShadow: `0 6px 18px ${c.accentGlow}`,
          transition: 'transform 150ms ease-out',
          '&:hover': {
            backgroundColor: `${c.accent} !important`,
            transform: 'scale(1.08)',
          },
        },
      },
      NDAlbumDetails: {
        cover: { borderRadius: radii.art },
        recordName: heading(800, '-0.03em'),
      },
      NDAlbumShow: { albumActions: actionBarStyle(c) },
      NDPlaylistDetails: {
        cover: { borderRadius: radii.art },
        title: heading(800, '-0.03em'),
      },
      NDPlaylistShow: { playlistActions: actionBarStyle(c) },
      NDDesktopArtistDetails: {
        cover: { borderRadius: radii.art },
        artistImage: { borderRadius: radii.art },
        artistName: heading(800, '-0.03em'),
      },
      NDMobileArtistDetails: {
        bgContainer: {
          background: `linear-gradient(to bottom, ${hexToRgba(c.background, 0.72)}, ${c.background})!important`,
        },
        cover: { borderRadius: radii.art },
        artistImage: { borderRadius: radii.art },
      },
    },
    player: {
      theme: mode,
      stylesheet: createVibrantPlayerStylesheet(c, radii),
    },
  }
}

export default createVibrantTheme
