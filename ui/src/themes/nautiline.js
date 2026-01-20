/**
 * Nautiline Theme for Navidrome
 * Light theme inspired by the Nautiline iOS app
 */

// ============================================
// CONFIGURATION
// ============================================

const ACCENT_COLOR = '#009688' // Material teal

// ============================================
// DESIGN TOKENS
// ============================================

const hexToRgb = (hex) => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : null
}

const rgb = hexToRgb(ACCENT_COLOR)
const rgba = (alpha) =>
  rgb ? `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})` : 'transparent'

const tokens = {
  colors: {
    accent: {
      main: ACCENT_COLOR,
      faded: rgba(0.1),
      hover: rgba(0.15),
    },
    background: {
      primary: '#FFFFFF',
      secondary: '#F5F5F7',
      tertiary: '#E5E5EA',
    },
    text: {
      primary: '#1A1A1A',
      secondary: '#8E8E93',
      tertiary: '#AEAEB2',
    },
    ui: {
      separator: 'rgba(0, 0, 0, 0.08)',
      shadow: 'rgba(0, 0, 0, 0.04)',
      glassBg: 'rgba(255, 255, 255, 0.72)',
    },
  },
  typography: {
    fontFamily: {
      base: [
        '-apple-system',
        'BlinkMacSystemFont',
        '"SF Pro Text"',
        '"Helvetica Neue"',
        'Arial',
        'sans-serif',
      ].join(','),
      heading: '"Unbounded", sans-serif',
    },
    fontFace: `
      @font-face {
        font-family: 'Unbounded';
        font-style: normal;
        font-weight: 300 800;
        font-display: swap;
        src: url('/fonts/Unbounded-Variable.woff2') format('woff2');
      }
    `,
  },
  spacing: {
    xs: '0.25rem',
    sm: '0.5rem',
    md: '0.75rem',
    lg: '1rem',
    xl: '1.5rem',
  },
  radii: {
    sm: '0.25rem',
    md: '0.5rem',
    lg: '0.625rem',
    xl: '0.75rem',
    full: '50%',
    pill: '1rem',
  },
  breakpoints: {
    xs: 599,
    sm: 600,
    md: 720,
    lg: 1280,
  },
  sizing: {
    cover: {
      sm: '14em',
      lg: '18em',
    },
    icon: '1.25rem',
    iconMinWidth: '2.5rem',
  },
  blur: '1.25rem',
}

const { colors, typography, spacing, radii, sizing, breakpoints } = tokens

// ============================================
// REUSABLE STYLE FACTORIES
// ============================================

const headingStyle = (weight, letterSpacing) => ({
  fontFamily: typography.fontFamily.heading,
  fontWeight: weight,
  ...(letterSpacing && { letterSpacing }),
})

const coverSizing = () => ({
  [`@media (min-width: ${breakpoints.sm}px)`]: {
    height: sizing.cover.sm,
    width: sizing.cover.sm,
    minWidth: sizing.cover.sm,
  },
  [`@media (min-width: ${breakpoints.lg}px)`]: {
    height: sizing.cover.lg,
    width: sizing.cover.lg,
    minWidth: sizing.cover.lg,
  },
})

const customTooltipStyle = () => ({
  display: 'inline',
  position: 'absolute',
  bottom: '100%',
  left: '50%',
  transform: 'translateX(-50%)',
  marginBottom: spacing.xs,
  fontSize: '0.75rem',
  whiteSpace: 'nowrap',
  backgroundColor: colors.text.primary,
  color: colors.background.primary,
  padding: `${spacing.xs} ${spacing.sm}`,
  borderRadius: radii.sm,
  zIndex: 9999,
})

const actionButtonsStyle = () => ({
  padding: `${spacing.lg} 0`,
  alignItems: 'center',
  '@global': {
    button: {
      border: '1px solid transparent',
      backgroundColor: colors.background.secondary,
      color: colors.text.secondary,
      margin: `0 ${spacing.sm}`,
      borderRadius: radii.full,
      minWidth: 0,
      padding: spacing.lg,
      position: 'relative',
      '&:hover': {
        backgroundColor: `${colors.background.tertiary} !important`,
        border: '1px solid transparent',
      },
    },
    'button:first-child:not(:only-child)': {
      [`@media screen and (max-width: ${breakpoints.md}px)`]: {
        transform: 'scale(1.5)',
        margin: spacing.lg,
        '&:hover': {
          transform: 'scale(1.6) !important',
        },
      },
      transform: 'scale(2)',
      margin: spacing.xl,
      minWidth: 0,
      padding: '0.3125rem',
      transition: 'transform .3s ease',
      background: colors.accent.main,
      color: '#fff',
      borderRadius: radii.full,
      border: 0,
      '&:hover': {
        transform: 'scale(2.1)',
        backgroundColor: `${colors.accent.main} !important`,
        border: 0,
      },
    },
    'button:only-child': {
      margin: spacing.xl,
    },
    'button:first-child>span:first-child': {
      padding: 0,
    },
    'button>span:first-child>span': {
      display: 'none',
    },
    'button:not(:first-child):hover>span:first-child>span':
      customTooltipStyle(),
    'button:not(:first-child)>span:first-child>svg': {
      color: colors.text.secondary,
    },
  },
})

const menuIconStyle = () => ({
  color: colors.text.primary,
  minWidth: sizing.iconMinWidth,
  '& svg': {
    fontSize: sizing.icon,
  },
})

const activeLinkStyle = {
  color: `${colors.accent.main} !important`,
  '& .MuiListItemIcon-root': {
    color: `${colors.accent.main} !important`,
  },
}

// ============================================
// THEME DEFINITION
// ============================================

// Note: !important declarations are required to override react-admin and third-party component styles
const NautilineTheme = {
  themeName: 'Nautiline',
  palette: {
    type: 'light',
    primary: {
      main: colors.accent.main,
      contrastText: '#FFFFFF',
    },
    secondary: {
      main: colors.accent.main,
      contrastText: '#FFFFFF',
    },
    background: {
      default: colors.background.primary,
      paper: colors.background.primary,
    },
    text: {
      primary: colors.text.primary,
      secondary: colors.text.secondary,
    },
    action: {
      active: colors.accent.main,
      hover: colors.accent.faded,
      selected: colors.accent.faded,
    },
  },
  typography: {
    fontFamily: typography.fontFamily.base,
    h1: headingStyle(700, '-0.02em'),
    h2: headingStyle(700, '-0.02em'),
    h3: headingStyle(600, '-0.01em'),
    h4: headingStyle(600),
    h5: headingStyle(600),
    h6: headingStyle(600),
    subtitle1: { fontWeight: 500 },
    subtitle2: { fontWeight: 500 },
    body1: { fontWeight: 400 },
    body2: { fontWeight: 400 },
    button: { fontWeight: 500, textTransform: 'none' },
  },
  shape: {
    borderRadius: radii.xl,
  },
  overrides: {
    MuiCssBaseline: {
      '@global': {
        '@font-face': {
          fontFamily: 'Unbounded',
          fontStyle: 'normal',
          fontWeight: '300 800',
          fontDisplay: 'swap',
          src: "url('/fonts/Unbounded-Variable.woff2') format('woff2')",
        },
        body: {
          backgroundColor: colors.background.primary,
        },
      },
    },
    MuiAppBar: {
      root: {
        boxShadow: 'none',
        borderBottom: `1px solid ${colors.ui.separator}`,
      },
      colorSecondary: {
        backgroundColor: colors.background.primary,
        color: colors.text.primary,
      },
    },
    MuiToolbar: {
      root: {
        backgroundColor: colors.background.primary,
      },
    },
    MuiPaper: {
      root: {
        backgroundColor: colors.background.primary,
      },
      elevation1: {
        boxShadow: `0 0.0625rem 0.1875rem ${colors.ui.shadow}`,
      },
      elevation2: {
        boxShadow: `0 0.125rem ${spacing.sm} ${colors.ui.shadow}`,
      },
    },
    MuiCard: {
      root: {
        backgroundColor: colors.background.primary,
        borderRadius: radii.xl,
        boxShadow: `0 0.125rem ${spacing.sm} ${colors.ui.shadow}`,
      },
    },
    MuiButton: {
      root: {
        borderRadius: radii.md,
        textTransform: 'none',
        fontWeight: 600,
      },
      contained: {
        boxShadow: 'none',
        '&:hover': { boxShadow: 'none' },
      },
      containedPrimary: {
        backgroundColor: colors.accent.main,
        '&:hover': {
          backgroundColor: colors.accent.main,
          filter: 'brightness(0.9)',
        },
      },
      text: {
        color: colors.accent.main,
      },
    },
    MuiIconButton: {
      root: {
        color: colors.text.primary,
        '&:hover': {
          backgroundColor: colors.accent.faded,
        },
      },
      colorPrimary: {
        color: colors.accent.main,
      },
      sizeSmall: {
        padding: spacing.md,
      },
    },
    MuiSvgIcon: {
      colorPrimary: {
        color: colors.accent.main,
      },
    },
    MuiCheckbox: {
      root: {
        color: 'rgba(0, 0, 0, 0.15)',
        '&$checked': {
          color: colors.accent.main,
        },
      },
    },
    MuiChip: {
      root: {
        backgroundColor: colors.background.secondary,
        color: colors.text.primary,
        borderRadius: radii.pill,
      },
      colorPrimary: {
        backgroundColor: colors.accent.faded,
        color: colors.accent.main,
      },
    },
    MuiTableRow: {
      root: {
        '&:hover': {
          backgroundColor: `${colors.accent.faded} !important`,
        },
      },
    },
    MuiTableCell: {
      root: {
        borderBottomColor: 'rgba(0, 0, 0, 0.04)',
      },
      head: {
        backgroundColor: colors.background.secondary,
        color: colors.text.secondary,
        fontWeight: 600,
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: '0.05em',
      },
      body: {
        color: colors.text.primary,
      },
    },
    MuiListItem: {
      root: {
        color: colors.text.primary,
        '&:hover': {
          backgroundColor: colors.accent.faded,
        },
        '&$selected': {
          backgroundColor: colors.accent.faded,
          color: colors.accent.main,
          '& .MuiListItemIcon-root': {
            color: colors.accent.main,
          },
          '&:hover': {
            backgroundColor: colors.accent.faded,
          },
        },
      },
      button: {
        color: colors.text.primary,
        '&:hover': {
          backgroundColor: colors.accent.faded,
          color: colors.text.primary,
        },
      },
    },
    MuiListItemIcon: {
      root: menuIconStyle(),
    },
    MuiListItemText: {
      primary: {
        color: 'inherit',
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '0.875rem',
        paddingTop: '4px',
        paddingBottom: '4px',
        paddingLeft: '10px',
        margin: '5px',
        borderRadius: radii.md,
        color: colors.text.primary,
      },
    },
    MuiDrawer: {
      paper: {
        backgroundColor: colors.background.primary,
        borderRight: `1px solid ${colors.ui.separator}`,
      },
    },
    MuiSlider: {
      root: {
        color: colors.accent.main,
      },
      track: {
        backgroundColor: colors.accent.main,
      },
      thumb: {
        backgroundColor: colors.accent.main,
        '&:hover': {
          boxShadow: `0 0 0 ${spacing.sm} ${colors.accent.faded}`,
        },
      },
      rail: {
        backgroundColor: colors.background.tertiary,
      },
    },
    MuiLinearProgress: {
      root: {
        backgroundColor: colors.background.tertiary,
        borderRadius: radii.sm,
      },
      bar: {
        backgroundColor: colors.accent.main,
        borderRadius: radii.sm,
      },
    },
    MuiTabs: {
      root: {
        borderBottom: `1px solid ${colors.ui.separator}`,
      },
      indicator: {
        backgroundColor: colors.accent.main,
        height: '0.1875rem',
        borderRadius: '0.1875rem 0.1875rem 0 0',
      },
    },
    MuiTab: {
      root: {
        textTransform: 'none',
        fontWeight: 500,
        fontFamily: typography.fontFamily.heading,
        '&$selected': {
          color: colors.accent.main,
          fontWeight: 600,
        },
      },
    },
    MuiInputBase: {
      root: {
        backgroundColor: colors.background.secondary,
        borderRadius: radii.lg,
      },
    },
    MuiOutlinedInput: {
      root: {
        borderRadius: radii.lg,
        '& $notchedOutline': {
          borderColor: colors.ui.separator,
        },
        '&:hover $notchedOutline': {
          borderColor: colors.text.tertiary,
        },
        '&$focused $notchedOutline': {
          borderColor: colors.accent.main,
          borderWidth: '0.125rem',
        },
      },
    },
    MuiFilledInput: {
      root: {
        backgroundColor: colors.background.secondary,
        borderRadius: radii.lg,
        '&:hover': {
          backgroundColor: colors.background.tertiary,
        },
        '&$focused': {
          backgroundColor: colors.background.secondary,
        },
      },
    },
    MuiFab: {
      primary: {
        backgroundColor: colors.accent.main,
        '&:hover': {
          backgroundColor: colors.accent.main,
          filter: 'brightness(0.9)',
        },
      },
    },
    MuiAvatar: {
      root: {
        borderRadius: radii.md,
      },
    },
    MuiRating: {
      iconFilled: {
        color: colors.accent.main,
      },
      iconHover: {
        color: colors.accent.main,
      },
    },
    MuiTooltip: {
      tooltip: {
        backgroundColor: colors.text.primary,
        color: colors.background.primary,
        fontSize: '0.75rem',
        padding: `${spacing.xs} ${spacing.sm}`,
        borderRadius: radii.sm,
      },
    },
    MuiBottomNavigation: {
      root: {
        backgroundColor: colors.ui.glassBg,
        backdropFilter: `blur(${tokens.blur})`,
        borderTop: `1px solid ${colors.ui.separator}`,
      },
    },
    MuiBottomNavigationAction: {
      root: {
        color: colors.text.secondary,
        '&$selected': {
          color: colors.accent.main,
        },
      },
      label: {
        fontFamily: typography.fontFamily.heading,
        fontSize: '0.65rem',
        '&$selected': {
          fontSize: '0.65rem',
        },
      },
    },
    NDAppBar: {
      root: {
        color: colors.text.primary,
      },
    },
    NDLogin: {
      main: {
        backgroundColor: colors.background.primary,
      },
      card: {
        backgroundColor: colors.background.primary,
        borderRadius: radii.pill,
        boxShadow: `0 ${spacing.xs} ${spacing.xl} ${colors.ui.shadow}`,
      },
    },
    NDAlbumGridView: {
      albumContainer: {
        borderRadius: radii.md,
        '& img': {
          borderRadius: radii.md,
        },
      },
      albumTitle: {
        fontWeight: 600,
        color: colors.text.primary,
      },
      albumSubtitle: {
        color: colors.text.secondary,
      },
      albumPlayButton: {
        backgroundColor: colors.accent.main,
        borderRadius: radii.full,
        boxShadow: `0 ${spacing.sm} ${spacing.sm} rgba(0, 0, 0, 0.15)`,
        padding: '0.35rem',
        transition: 'padding .3s ease',
        '&:hover': {
          backgroundColor: `${colors.accent.main} !important`,
          padding: '0.45rem',
        },
      },
    },
    NDAlbumDetails: {
      root: {
        [`@media (max-width: ${breakpoints.xs}px)`]: {
          padding: '0.7em',
          width: '100%',
          minWidth: 'unset',
        },
      },
      cardContents: {
        [`@media (max-width: ${breakpoints.xs}px)`]: {
          flexDirection: 'column',
          alignItems: 'center',
        },
      },
      details: {
        [`@media (max-width: ${breakpoints.xs}px)`]: {
          width: '100%',
        },
      },
      cover: {
        borderRadius: radii.md,
      },
      coverParent: {
        marginRight: spacing.xl,
        [`@media (max-width: ${breakpoints.xs}px)`]: {
          width: '100%',
          height: 'auto',
          minWidth: 'unset',
          aspectRatio: '1',
          marginRight: 0,
          marginBottom: spacing.lg,
        },
        ...coverSizing(),
      },
      recordName: {
        fontSize: '1.75rem',
        fontWeight: 700,
        marginBottom: '0.15rem',
      },
      recordArtist: {
        marginBottom: spacing.md,
      },
      recordMeta: {
        marginBottom: spacing.sm,
      },
      genreList: {
        marginTop: spacing.md,
      },
      loveButton: {
        marginLeft: spacing.sm,
      },
    },
    NDAlbumShow: {
      albumActions: actionButtonsStyle(),
    },
    NDPlaylistShow: {
      playlistActions: actionButtonsStyle(),
    },
    NDSubMenu: {
      icon: menuIconStyle(),
      menuHeader: {
        color: colors.text.primary,
        '& .MuiTypography-root': {
          color: colors.text.primary,
        },
      },
      actionIcon: {
        marginLeft: spacing.sm,
      },
    },
    RaMenuItemLink: {
      root: {
        color: `${colors.text.primary} !important`,
        '& .MuiListItemIcon-root': menuIconStyle(),
        '&[class*="makeStyles-active"]': activeLinkStyle,
      },
      active: activeLinkStyle,
    },
    NDDesktopArtistDetails: {
      root: {
        [`@media (min-width: ${breakpoints.sm}px)`]: {
          padding: '1em',
        },
        [`@media (min-width: ${breakpoints.lg}px)`]: {
          padding: '1em',
        },
      },
      cover: {
        borderRadius: radii.md,
        ...coverSizing(),
      },
      artistImage: {
        borderRadius: radii.md,
        marginRight: spacing.xl,
        [`@media (min-width: ${breakpoints.sm}px)`]: {
          height: sizing.cover.sm,
          width: sizing.cover.sm,
          minWidth: sizing.cover.sm,
          maxHeight: sizing.cover.sm,
          minHeight: sizing.cover.sm,
        },
        [`@media (min-width: ${breakpoints.lg}px)`]: {
          height: sizing.cover.lg,
          width: sizing.cover.lg,
          minWidth: sizing.cover.lg,
          maxHeight: sizing.cover.lg,
          minHeight: sizing.cover.lg,
        },
      },
      artistName: {
        fontSize: '1.75rem',
        fontWeight: 700,
        marginBottom: spacing.sm,
      },
    },
    NDMobileArtistDetails: {
      cover: {
        borderRadius: radii.md,
      },
      artistImage: {
        borderRadius: radii.md,
      },
    },
    RaList: {
      content: {
        overflow: 'visible',
      },
    },
    RaBulkActionsToolbar: {
      topToolbar: {
        backgroundColor: 'transparent',
        boxShadow: 'none',
        padding: spacing.sm,
        '@global': {
          button: {
            border: '1px solid transparent',
            backgroundColor: colors.background.secondary,
            color: colors.text.secondary,
            margin: `0 ${spacing.xs}`,
            borderRadius: radii.full,
            minWidth: 0,
            padding: spacing.sm,
            position: 'relative',
            '&:hover': {
              backgroundColor: `${colors.background.tertiary} !important`,
              border: '1px solid transparent',
            },
          },
          'button>span:first-child>span': {
            display: 'none',
          },
          'button:hover>span:first-child>span': customTooltipStyle(),
          'button>span:first-child>svg': {
            color: colors.text.secondary,
          },
        },
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        backgroundColor: colors.accent.faded,
      },
    },
  },
  player: {
    theme: 'light',
    stylesheet: `
      @font-face {
        font-family: 'Unbounded';
        font-style: normal;
        font-weight: 300 800;
        font-display: swap;
        src: url('/fonts/Unbounded-Variable.woff2') format('woff2');
      }

      .react-jinke-music-player-main {
        background-color: ${colors.background.primary} !important;
        font-family: ${typography.fontFamily.base} !important;
      }

      .react-jinke-music-player-main .music-player-panel {
        background-color: ${colors.ui.glassBg} !important;
        backdrop-filter: blur(${tokens.blur}) !important;
        -webkit-backdrop-filter: blur(${tokens.blur}) !important;
        border-top: 1px solid ${colors.ui.separator} !important;
        box-shadow: 0 -0.125rem 1.25rem rgba(0, 0, 0, 0.06) !important;
      }

      .react-jinke-music-player-main svg {
        color: ${colors.text.primary} !important;
      }

      .react-jinke-music-player-main svg:hover {
        color: ${colors.accent.main} !important;
      }

      .react-jinke-music-player-main .rc-slider-track,
      .react-jinke-music-player-main .rc-slider-handle {
        background-color: ${colors.accent.main} !important;
      }

      .react-jinke-music-player-main .rc-slider-handle {
        border-color: ${colors.accent.main} !important;
      }

      .react-jinke-music-player-main .rc-slider-rail {
        background-color: ${colors.background.secondary} !important;
      }

      .react-jinke-music-player-main .rc-slider {
        height: 4px !important;
      }

      .react-jinke-music-player-main .rc-slider-rail,
      .react-jinke-music-player-main .rc-slider-track {
        height: 4px !important;
        border-radius: 2px !important;
      }

      .react-jinke-music-player-main .rc-slider-handle {
        width: 12px !important;
        height: 12px !important;
        margin-top: -4px !important;
      }

      .react-jinke-music-player-main .audio-lists-panel,
      .react-jinke-music-player-main .audio-lists-panel-content {
        background-color: ${colors.background.primary} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item {
        background-color: transparent !important;
        color: ${colors.text.primary} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover {
        background-color: ${colors.accent.faded} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item.playing {
        background-color: ${colors.accent.faded} !important;
        color: ${colors.accent.main} !important;
      }

      .react-jinke-music-player-main .lyric-btn-active,
      .react-jinke-music-player-main .play-mode-title {
        color: ${colors.accent.main} !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-title {
        color: ${colors.text.primary} !important;
        font-weight: 600 !important;
        font-family: ${typography.fontFamily.heading} !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .music-player-controller .music-player-info .music-player-artist {
        color: ${colors.text.secondary} !important;
      }

      .react-jinke-music-player-main.mini-player {
        background-color: ${colors.ui.glassBg} !important;
        backdrop-filter: blur(${tokens.blur}) !important;
        -webkit-backdrop-filter: blur(${tokens.blur}) !important;
        border-radius: ${radii.xl} !important;
        box-shadow: 0 ${spacing.xs} 1.25rem rgba(0, 0, 0, 0.08) !important;
      }


      .MuiTypography-h1,
      .MuiTypography-h2,
      .MuiTypography-h3,
      .MuiTypography-h4,
      .MuiTypography-h5,
      .MuiTypography-h6 {
        font-family: ${typography.fontFamily.heading} !important;
      }
    `,
  },
}

export default NautilineTheme
