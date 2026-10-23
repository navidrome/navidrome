import stylesheet from './cave.css.js'

// Near-black background with a single orange accent, based on the colors of
// https://acave.pt (A Cave do Entulho)
const background = '#0b0b0d'
const surface = '#151518'
const text = '#f2f2f0'
const textMuted = '#b7b7b7'
const accent = '#ff4d1a'
const accentDark = '#e84518'

export default {
  themeName: 'Cave',
  palette: {
    primary: {
      main: accent,
      contrastText: background,
    },
    secondary: {
      main: accent,
      contrastText: background,
    },
    background: {
      default: background,
      paper: surface,
    },
    text: {
      primary: text,
      secondary: textMuted,
    },
    type: 'dark',
  },
  overrides: {
    MuiAppBar: {
      positionFixed: {
        backgroundColor: `${background} !important`,
        boxShadow: 'none',
        borderBottom: `1px solid ${accent}`,
      },
      colorSecondary: {
        color: text,
      },
    },
    MuiDrawer: {
      root: {
        background: background,
      },
    },
    MuiToolbar: {
      root: {
        background: 'transparent !important',
      },
    },
    MuiButton: {
      textPrimary: {
        color: accent,
      },
      containedPrimary: {
        color: background,
        backgroundColor: accent,
        '&:hover': {
          backgroundColor: accentDark,
          '@media (hover: none)': {
            backgroundColor: accent,
          },
        },
      },
    },
    MuiListItemIcon: {
      root: {
        color: textMuted,
      },
    },
    MuiTableBody: {
      root: {
        '&>tr:nth-child(odd)': {
          background: 'rgba(255, 255, 255, 0.025)',
        },
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: '0 none !important',
      },
      head: {
        color: `${textMuted} !important`,
      },
    },
    MuiPaper: {
      elevation1: {
        boxShadow: 'none',
      },
    },
    NDAlbumGridView: {
      albumName: {
        color: text,
      },
      albumArtistName: {
        color: textMuted,
      },
      albumPlayButton: {
        color: accent,
      },
    },
    NDLogin: {
      systemNameLink: {
        color: accent,
      },
      welcome: {
        color: text,
      },
      card: {
        minWidth: 300,
        backgroundColor: `${background}e6`,
      },
      icon: {
        filter: 'hue-rotate(170deg) saturate(1.2)',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background: background,
      },
    },
    RaPaginationActions: {
      currentPageButton: {
        border: `2px solid ${accent}`,
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
