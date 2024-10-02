import stylesheet from './nuclear.css.js'

const nukeCol = {
  primary: '#1d2021',
  secondary: '#282828',
  accent: '#32302f',
  text: '#ebdbb2',
  textAlt: '#bdae93',
  icon: '#b8bb26',
  link: '#c44129',
  border: '#a89984',
}

export default {
  themeName: 'Nuclear',
  palette: {
    primary: {
      main: nukeCol['primary'],
    },
    secondary: {
      main: nukeCol['secondary'],
    },
    background: {
      default: nukeCol['primary'],
    },
    text: {
      primary: nukeCol['text'],
      secondary: nukeCol['text'],
    },
    type: 'dark',
  },
  overrides: {
    MuiTypography: {
      root: {
        color: nukeCol['text'],
      },
      colorPrimary: {
        color: nukeCol['text'],
      },
    },
    MuiPaper: {
      root: {
        backgroundColor: nukeCol['secondary'],
      },
    },
    MuiFormGroup: {
      root: {
        color: nukeCol['text'],
      },
    },
    NDAlbumGridView: {
      albumName: {
        marginTop: '0.5rem',
        fontWeight: 700,
        textTransform: 'none',
        color: nukeCol['text'],
      },
      albumSubtitle: {
        color: nukeCol['textAlt'],
      },
    },
    MuiAppBar: {
      colorSecondary: {
        color: nukeCol['text'],
      },
      positionFixed: {
        backgroundColor: nukeCol['primary'],
        boxShadow:
          'rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px',
      },
    },
    MuiButton: {
      root: {
        border: '1px solid transparent',
        '&:hover': {
          backgroundColor: nukeCol['accent'],
        },
      },
      label: {
        color: nukeCol['text'],
      },
      contained: {
        boxShadow: 'none',
        '&:hover': {
          boxShadow: 'none',
        },
      },
    },
    MuiChip: {
      root: {
        backgroundColor: nukeCol['accent'],
      },
      label: {
        color: nukeCol['icon'],
      },
    },
    RaLink: {
      link: {
        color: nukeCol['link'],
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: 'none',
        color: nukeCol['text'],
        padding: '10px !important',
      },
      head: {
        fontSize: '0.75rem',
        textTransform: 'uppercase',
        letterSpacing: 1.2,
        backgroundColor: nukeCol['accent'],
        color: nukeCol['text'],
      },
      body: {
        color: nukeCol['text'],
      },
    },
    MuiInput: {
      root: {
        color: nukeCol['text'],
      },
    },
    MuiFormLabel: {
      root: {
        '&$focused': {
          color: nukeCol['text'],
          fontWeight: 'bold',
        },
      },
    },
    MuiOutlinedInput: {
      notchedOutline: {
        borderColor: nukeCol['border'],
      },
    },
    //Icons
    MuiIconButton: {
      label: {
        color: nukeCol['icon'],
      },
    },
    MuiListItemIcon: {
      root: {
        color: nukeCol['icon'],
      },
    },
    MuiSelect: {
      icon: {
        color: nukeCol['icon'],
      },
    },
    MuiSvgIcon: {
      root: {
        color: nukeCol['icon'],
      },
      colorDisabled: {
        color: nukeCol['icon'],
      },
    },
    MuiSwitch: {
      colorPrimary: {
        '&$checked + $track': {
          backgroundColor: '#f9f5d7',
        },
      },
      track: {
        backgroundColor: '#665c54',
      },
    },
    RaButton: {
      smallIcon: {
        color: nukeCol['icon'],
      },
    },
    RaDatagrid: {
      headerCell: {
        backgroundColor: nukeCol['accent'],
      },
    },
    //Login Screen
    NDLogin: {
      systemNameLink: {
        color: nukeCol['text'],
      },
      card: {
        minWidth: 300,
        backgroundColor: nukeCol['secondary'],
      },
      button: {
        boxShadow: '3px 3px 5px #000000a3',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52 52 52 / 72%), rgb(48 48 48))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
