import NautilineTheme from './nautiline'

const colors = {
  accent: '#7C3AED',
  accentBright: '#8B5CF6',
  accentSoft: 'rgba(124, 58, 237, 0.16)',
  accentHover: 'rgba(124, 58, 237, 0.24)',
  danger: '#F44336',
  background: '#000000',
  surface: '#1B1720',
  elevated: '#27202D',
  text: '#F7F5FA',
  textSecondary: '#A9A4B0',
  textTertiary: '#77717F',
  separator: 'rgba(165, 125, 190, 0.16)',
}

const appleFont = [
  '-apple-system',
  'BlinkMacSystemFont',
  '"SF Pro Display"',
  '"SF Pro Text"',
  '"Helvetica Neue"',
  'Arial',
  'sans-serif',
].join(', ')

const replacements = [
  [/"Unbounded", sans-serif/g, appleFont],
  [/#009688/gi, colors.accent],
  [/rgba\(0, 150, 136, 0\.1\)/g, colors.accentSoft],
  [/rgba\(0, 150, 136, 0\.15\)/g, colors.accentHover],
  [/#FFFFFF/gi, colors.background],
  [/#F5F5F7/gi, colors.surface],
  [/#E5E5EA/gi, colors.elevated],
  [/#1A1A1A/gi, colors.text],
  [/#8E8E93/gi, colors.textSecondary],
  [/#AEAEB2/gi, colors.textTertiary],
  [/rgba\(0, 0, 0, 0\.08\)/g, colors.separator],
  [/rgba\(0, 0, 0, 0\.04\)/g, 'rgba(0, 0, 0, 0.28)'],
  [/rgba\(255, 255, 255, 0\.72\)/g, 'rgba(27, 23, 32, 0.9)'],
]

const recolor = (value) => {
  if (typeof value === 'string') {
    return replacements.reduce(
      (result, [pattern, replacement]) => result.replace(pattern, replacement),
      value,
    )
  }

  if (Array.isArray(value)) return value.map(recolor)

  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value).map(([key, child]) => [key, recolor(child)]),
    )
  }

  return value
}

const base = recolor(NautilineTheme)

const formButtonBase = {
  minWidth: 96,
  minHeight: 40,
  padding: '8px 14px',
  borderRadius: '8px !important',
  boxShadow: 'none',
}

const formSaveButton = {
  ...formButtonBase,
  color: '#FFFFFF !important',
  backgroundColor: `${colors.accent} !important`,
  border: '1px solid transparent',
  '&:hover': {
    backgroundColor: `${colors.accentBright} !important`,
    boxShadow: 'none',
  },
  '&.Mui-disabled': {
    color: `${colors.textTertiary} !important`,
    backgroundColor: `${colors.elevated} !important`,
    borderColor: colors.separator,
  },
}

const ShelvTheme = {
  ...base,
  themeName: 'Shelv',
  typography: {
    ...base.typography,
    fontFamily: appleFont,
    h1: { ...base.typography.h1, fontFamily: appleFont, fontWeight: 700 },
    h2: { ...base.typography.h2, fontFamily: appleFont, fontWeight: 700 },
    h3: { ...base.typography.h3, fontFamily: appleFont, fontWeight: 650 },
    h4: { ...base.typography.h4, fontFamily: appleFont, fontWeight: 650 },
    h5: { ...base.typography.h5, fontFamily: appleFont, fontWeight: 600 },
    h6: { ...base.typography.h6, fontFamily: appleFont, fontWeight: 600 },
    button: {
      ...base.typography.button,
      fontFamily: appleFont,
      fontWeight: 600,
      letterSpacing: '-0.01em',
    },
  },
  palette: {
    ...base.palette,
    type: 'dark',
    primary: { main: colors.accent, contrastText: '#FFFFFF' },
    secondary: { main: colors.accentBright, contrastText: '#FFFFFF' },
    background: { default: colors.background, paper: colors.surface },
    text: { primary: colors.text, secondary: colors.textSecondary },
    divider: colors.separator,
    action: {
      ...base.palette.action,
      active: colors.accentBright,
      hover: colors.accentSoft,
      selected: colors.accentHover,
    },
  },
  shape: { borderRadius: 10 },
  overrides: {
    ...base.overrides,
    MuiCssBaseline: {
      ...base.overrides.MuiCssBaseline,
      '@global': {
        ...base.overrides.MuiCssBaseline['@global'],
        body: {
          backgroundColor: colors.background,
        },
        '*': {
          scrollbarColor: `${colors.elevated} transparent`,
        },
      },
    },
    MuiButton: {
      ...base.overrides.MuiButton,
      root: {
        ...base.overrides.MuiButton.root,
        borderRadius: 6,
      },
    },
    RaSaveButton: {
      ...base.overrides.RaSaveButton,
      button: {
        ...base.overrides.RaSaveButton?.button,
        ...formSaveButton,
      },
    },
    NDPluginShow: {
      ...base.overrides.NDPluginShow,
      saveButton: {
        ...base.overrides.NDPluginShow?.saveButton,
        ...formSaveButton,
      },
    },
    MuiIconButton: {
      ...base.overrides.MuiIconButton,
      root: {
        ...base.overrides.MuiIconButton.root,
        borderRadius: 999,
        transition: 'background-color 120ms ease, color 120ms ease',
        '&:hover': {
          backgroundColor: colors.accentSoft,
          color: colors.accentBright,
        },
      },
    },
    MuiCheckbox: {
      ...base.overrides.MuiCheckbox,
      root: {
        ...base.overrides.MuiCheckbox.root,
        color: 'rgba(165, 125, 190, 0.68)',
        '&:hover': {
          color: colors.text,
          backgroundColor: colors.accentSoft,
        },
        '&$checked': {
          color: colors.accentBright,
        },
      },
    },
    MuiPaper: {
      ...base.overrides.MuiPaper,
      root: {
        ...base.overrides.MuiPaper.root,
        backgroundColor: colors.surface,
        backgroundImage: 'none',
      },
    },
    MuiPopover: {
      ...base.overrides.MuiPopover,
      root: {
        ...base.overrides.MuiPopover?.root,
        '&[id="menu-appbar"], &[id="panel-activity"], &[id="panel-nowplaying"]':
          {
            '& .MuiPopover-paper': {
              backgroundColor: colors.surface,
              borderRadius: 10,
              overflow: 'hidden',
            },
            '& .MuiCard-root': {
              backgroundColor: 'transparent',
              borderRadius: 0,
              boxShadow: 'none',
            },
            '& .MuiCardContent-root, & .MuiCardActions-root': {
              backgroundColor: 'transparent',
            },
          },
      },
    },
    MuiTableCell: {
      ...base.overrides.MuiTableCell,
      root: {
        ...base.overrides.MuiTableCell.root,
        borderBottomColor: colors.separator,
      },
      head: {
        ...base.overrides.MuiTableCell.head,
        backgroundColor: colors.surface,
        color: colors.textSecondary,
      },
    },
    MuiTableRow: {
      ...base.overrides.MuiTableRow,
      root: {
        ...base.overrides.MuiTableRow.root,
        backgroundColor: colors.surface,
        '&:hover': {
          backgroundColor: `${colors.elevated} !important`,
        },
      },
    },
    RaDatagrid: {
      ...base.overrides.RaDatagrid,
      rowCell: {
        ...base.overrides.RaDatagrid?.rowCell,
        '&&': {
          paddingTop: '14px !important',
          paddingBottom: '14px !important',
        },
      },
    },
    RaList: {
      ...base.overrides.RaList,
      root: {
        ...base.overrides.RaList?.root,
        backgroundColor: colors.surface,
        border: `1px solid ${colors.separator}`,
        borderRadius: 14,
        overflow: 'hidden',
        '& > .MuiBox-root:last-child': {
          paddingBottom: 24,
        },
      },
      toolbar: {
        ...base.overrides.RaList?.toolbar,
        backgroundColor: colors.surface,
        alignItems: 'center',
        border: `1px solid ${colors.separator}`,
        borderBottom: 0,
        borderRadius: '14px 14px 0 0',
        boxSizing: 'border-box',
        '& + div > .MuiCard-root': {
          borderLeft: `1px solid ${colors.separator}`,
          borderRight: `1px solid ${colors.separator}`,
        },
        '& ~ .MuiTablePagination-root': {
          borderLeft: `1px solid ${colors.separator}`,
          borderRight: `1px solid ${colors.separator}`,
          borderBottom: `1px solid ${colors.separator}`,
          borderRadius: '0 0 14px 14px',
          overflow: 'hidden',
        },
        '& ~ .MuiCardContent-root': {
          minHeight: 80,
          padding: '20px 24px !important',
          boxSizing: 'border-box',
          backgroundColor: colors.surface,
          borderLeft: `1px solid ${colors.separator}`,
          borderRight: `1px solid ${colors.separator}`,
          borderBottom: `1px solid ${colors.separator}`,
          borderRadius: '0 0 14px 14px',
          color: colors.textSecondary,
          '& .MuiTypography-root': {
            fontSize: '1rem',
            fontWeight: 400,
            textAlign: 'left',
          },
        },
      },
      content: {
        ...base.overrides.RaList.content,
        marginTop: 0,
        backgroundColor: colors.surface,
        borderRadius: 0,
        boxShadow: 'none',
        overflow: 'hidden',
      },
    },
    RaEmpty: {
      ...base.overrides.RaEmpty,
      message: {
        ...base.overrides.RaEmpty?.message,
        '&:last-child': {
          paddingBottom: 24,
        },
      },
      toolbar: {
        ...base.overrides.RaEmpty?.toolbar,
        paddingBottom: 24,
      },
    },
    RaListToolbar: {
      ...base.overrides.RaListToolbar,
      toolbar: {
        ...base.overrides.RaListToolbar?.toolbar,
        backgroundColor: colors.surface,
        paddingLeft: '16px !important',
        paddingRight: '16px !important',
        borderBottom: `1px solid ${colors.separator}`,
      },
      actions: {
        ...base.overrides.RaListToolbar?.actions,
        backgroundColor: colors.surface,
      },
    },
    RaSimpleList: {
      ...base.overrides.RaSimpleList,
      link: {
        ...base.overrides.RaSimpleList?.link,
        '& .MuiListItem-root': {
          paddingTop: 16,
          paddingBottom: 16,
        },
      },
    },
    RaArtistSimpleList: {
      ...base.overrides.RaArtistSimpleList,
      listItem: {
        ...base.overrides.RaArtistSimpleList?.listItem,
        padding: 16,
      },
      rightIcon: {
        ...base.overrides.RaArtistSimpleList?.rightIcon,
        top: '50%',
      },
    },
    MuiTablePagination: {
      ...base.overrides.MuiTablePagination,
      root: {
        ...base.overrides.MuiTablePagination?.root,
        backgroundColor: colors.surface,
        borderTop: `1px solid ${colors.separator}`,
        borderBottom: 0,
      },
    },
    RaToolbar: {
      ...base.overrides.RaToolbar,
      toolbar: {
        ...base.overrides.RaToolbar?.toolbar,
        backgroundColor: 'transparent',
        boxShadow: 'none',
        '& .MuiButton-root': {
          ...formButtonBase,
          fontWeight: 600,
          transition:
            'background-color 120ms ease, border-color 120ms ease, color 120ms ease',
        },
        '& .MuiButton-root .MuiButton-label svg': {
          fontSize: '18px !important',
        },
        '& .ra-delete-button': {
          color: '#FFFFFF !important',
          backgroundColor: `${colors.danger} !important`,
          border: '1px solid transparent',
          '&:hover': {
            color: '#FFFFFF !important',
            backgroundColor: '#C62828 !important',
            borderColor: 'transparent',
          },
          '&.Mui-disabled': {
            color: 'rgba(255, 92, 87, 0.38) !important',
            backgroundColor: `${colors.elevated} !important`,
            borderColor: colors.separator,
          },
        },
      },
      desktopToolbar: {
        ...base.overrides.RaToolbar?.desktopToolbar,
        width: 'fit-content',
        minHeight: 0,
        padding: '0 16px !important',
        gap: 12,
      },
      defaultToolbar: {
        ...base.overrides.RaToolbar?.defaultToolbar,
        justifyContent: 'flex-start',
        gap: 12,
      },
      mobileToolbar: {
        ...base.overrides.RaToolbar?.mobileToolbar,
        backgroundColor: colors.surface,
      },
    },
    RaAppBar: {
      ...base.overrides.RaAppBar,
      title: {
        ...base.overrides.RaAppBar?.title,
        color: colors.accentBright,
      },
    },
    NDPlaylistShow: {
      ...base.overrides.NDPlaylistShow,
      playlistActions: {
        ...base.overrides.NDPlaylistShow?.playlistActions,
        padding: '0 8px',
        alignItems: 'center',
        '& > div': {
          justifyContent: 'space-between',
          alignItems: 'center',
        },
        '& > div > div': {
          display: 'flex',
          alignItems: 'center',
        },
        '@global': {
          ...base.overrides.NDPlaylistShow?.playlistActions?.['@global'],
          button: {
            ...base.overrides.NDPlaylistShow?.playlistActions?.['@global']
              ?.button,
            margin: '0 4px',
            padding: 8,
          },
          'button:first-child:not(:only-child)': {
            ...base.overrides.NDPlaylistShow?.playlistActions?.['@global']?.[
              'button:first-child:not(:only-child)'
            ],
            transform: 'scale(1.35)',
            margin: '4px 10px',
            padding: 4,
            '@media screen and (max-width: 720px)': {
              transform: 'scale(1.25)',
              margin: '4px 8px',
              '&:hover': {
                transform: 'scale(1.3) !important',
              },
            },
            '&:hover': {
              ...base.overrides.NDPlaylistShow?.playlistActions?.['@global']?.[
                'button:first-child:not(:only-child)'
              ]?.['&:hover'],
              transform: 'scale(1.42) !important',
            },
          },
          'button:only-child': {
            ...base.overrides.NDPlaylistShow?.playlistActions?.['@global']?.[
              'button:only-child'
            ],
            margin: 4,
            padding: 4,
            backgroundColor: 'transparent',
          },
        },
      },
    },
    MuiAppBar: {
      ...base.overrides.MuiAppBar,
      root: {
        ...base.overrides.MuiAppBar.root,
        backgroundColor: 'rgba(0, 0, 0, 0.9)',
        backdropFilter: 'blur(16px)',
        borderBottom: `1px solid ${colors.separator}`,
      },
      colorSecondary: {
        backgroundColor: 'rgba(0, 0, 0, 0.9)',
        color: colors.text,
      },
    },
    MuiToolbar: {
      root: { backgroundColor: 'transparent' },
    },
    MuiDialog: {
      ...base.overrides.MuiDialog,
      paper: {
        ...base.overrides.MuiDialog?.paper,
        borderRadius: 12,
        border: `1px solid ${colors.separator}`,
        boxShadow: '0 18px 48px rgba(0, 0, 0, 0.38)',
        '& #config-panel .MuiButton-root + .MuiButton-root': {
          marginLeft: 12,
        },
      },
    },
    MuiTooltip: {
      ...base.overrides.MuiTooltip,
      tooltip: {
        ...base.overrides.MuiTooltip?.tooltip,
        borderRadius: 8,
        backgroundColor: colors.elevated,
        color: colors.text,
        border: `1px solid ${colors.separator}`,
      },
    },
  },
  player: {
    ...base.player,
    theme: 'dark',
    stylesheet: `${base.player.stylesheet}
      .react-jinke-music-player-main .music-player-panel .player-content svg {
        font-size: 28px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .play-btn svg,
      .react-jinke-music-player-main .music-player-panel .player-content .play-btn:hover svg,
      .react-jinke-music-player-mobile-toggle .play-btn svg {
        color: ${colors.accent} !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .play-btn svg {
        font-size: 36px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .lyric-btn svg {
        font-size: 22px !important;
      }

      .react-jinke-music-player-main .music-player-panel .player-content .audio-lists-btn {
        min-width: auto !important;
        padding: 0 4px !important;
        background-color: transparent !important;
        box-shadow: none !important;
      }

      .react-jinke-music-player-main .audio-lists-panel,
      .react-jinke-music-player-main .audio-lists-panel-content,
      .audio-lists-panel,
      .audio-lists-panel-content {
        background-color: ${colors.elevated} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-header,
      .audio-lists-panel-header {
        background-color: #30263A !important;
        border-bottom: 1px solid ${colors.separator} !important;
        box-shadow: none !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item,
      .audio-lists-panel-content .audio-item {
        background-color: ${colors.surface} !important;
        border-bottom: 1px solid ${colors.separator} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover,
      .audio-lists-panel-content .audio-item:hover {
        background-color: ${colors.accentSoft} !important;
      }

      .react-jinke-music-player-main .audio-lists-panel-content .audio-item.playing,
      .audio-lists-panel-content .audio-item.playing {
        background-color: ${colors.accentHover} !important;
        color: ${colors.accentBright} !important;
      }
    `,
  },
}

export default ShelvTheme
