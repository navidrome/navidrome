import stylesheet from './nutball.css.js'

export default {
  themeName: 'Nutball',
  palette: {
    primary: {
      main: '#80ea00',
      light: '#fff',
    },
    secondary: {
      main: '#80ea00',
      contrastText: '#fff',
    },
  },
  typography: {
    fontFamily: 'monospace',
    h6: {
      fontSize: '1rem',
    },
    h4: {
      fontSize: '1.2rem',
    },
    h1: {
      fontSize: '1.4rem',
    },
    body: {
      fontFamily: 'monospace',
    },
  },
  overrides: {
    MuiAppBar: {
      root: {
        borderBottom: '1px solid black',
      },
      colorSecondary: {
        color: 'black',
        backgroundColor: 'white',
      },
    },
    MuiPaper: {
      elevation1: {
        boxShadow: 'none',
      },
      elevation4: {
        boxShadow: 'none',
      },
      elevation6: {
        boxShadow: 'none',
      },
      elevation8: {
        boxShadow: 'none',
        border: '1px solid black',
      },
      elevation16: {
        boxShadow: 'none',
        borderRight: '1px solid grey!important',
      },
      elevation24: {
        boxShadow: 'none',
        border: '1px solid black',
      },
    },
    MuiButton: {
      root: {
        color: '#80ea00',
        border: '1px solid rgba(0, 0, 0, 0.23)',
        transition: 'none',
        '&[aria-label="Grid"]': {
          width: '50%',
          marginLeft: '15px',
          marginRight: '2px',
          marginBottom: '10px',
          '& .MuiButton-label': {
            justifyContent: 'center',
          },
        },
        '&[aria-label="Table"]': {
          width: '50%',
          marginRight: '15px',
          marginLeft: '2px',
          marginBottom: '10px',
          '& .MuiButton-label': {
            justifyContent: 'center',
          },
        },
      },
      textPrimary: {
        color: 'rgba(0,0,0,.57)',
        '&:hover': {
          borderColor: 'black',
          backgroundColor: '#eaeaea',
        },
        '&[aria-label="Grid"]': {
          color: 'black',
          borderColor: 'black!important',
        },
        '&[aria-label="Table"]': {
          color: 'black',
          borderColor: 'black!important',
        },
      },
      textSecondary: {
        color: 'rgba(0,0,0,.57)',
        '&:hover': {
          borderColor: 'black',
          backgroundColor: '#eaeaea',
        },
        '&[aria-label="Grid"]': {
          color: 'grey',
          borderColor: 'grey!important',
        },
        '&[aria-label="Table"]': {
          color: 'grey',
          borderColor: 'grey!important',
        },
      },
      label: {
        '& svg': {
          display: 'none',
        },
        '& span': {
          paddingLeft: '0',
        },
      },
      contained: {
        boxShadow: 'none',
        '&:hover': {
          boxShadow: 'none',
        },
      },
    },
    MuiButtonGroup: {
      groupedTextHorizontal: {
        justifyContent: 'flex-start',
        margin: '0 .5rem',
        '& button': {
          width: '25%',
        },
      },
      groupedTextPrimary: {
        '&:not(:last-child)': {
          border: 'none',
        },
      },
    },
    MuiIconButton: {
      root: {
        '&[aria-label="Settings"]': {
          padding: '12px!important',
          marginRight: '-9px!important',
        },
      },
    },
    MuiSwitch: {
      thumb: {
        color: '#eaeaea',
        boxShadow: 'none',
        borderRadius: '0',
      },
      track: {
        borderRadius: '0',
      },
      switchBase: {
        color: '#eaeaea',
      },
    },
    MuiCheckbox: {
      root: {
        '& svg': {
          width: '.8em',
        },
      },
    },
    PrivateSwitchBase: {
      root: {
        padding: '8px 8px',
      },
    },
    RaButton: {
      button: {
        marginRight: '10px',
        lineHeight: 'normal',
      },
    },
    MuiMenu: {
      list: {
        '& p': {
          fontSize: '.85rem',
        },
        '& p:first-of-type': {
          margin: '6px 1rem',
        },
        '& li:has(span.MuiCheckbox-root)': {
          marginLeft: '-10px',
        },
        '& span.MuiCheckbox-root .MuiSvgIcon-root': {
          width: '.75em',
        },
      },
    },
    MuiMenuItem: {
      root: {
        fontSize: '.85rem',
        minHeight: 'inherit',
        '&[aria-label="Clear value"]:before': {
          display: 'block',
          content: "'(any)'",
        },
      },
    },
    MuiListItem: {
      button: {
        '& span.MuiCheckbox-root': {
          padding: '0px 8px',
        },
      },
    },
    MuiTooltip: {
      tooltip: {
        backgroundColor: 'rgb(117 117 117)',
      },
    },
	MuiCircularProgress: {
		root: {
			color: '#80ea00!important'
		},
	},
    MuiAvatar: {
      img: {
        borderRadius: '5px',
      },
    },
    MuiFab: {
      root: {
        boxShadow: 'none',
      },
    },
    MuiTableHead: {
      root: {
        boxShadow: 'none!important',
      },
    },
    MuiTableCell: {
      root: {
        borderBottom: 'none',
      },
      sizeSmall: {
        '&:last-child': {
          textAlign: 'right',
        },
        '&:last-child:is(th)': {
          paddingRight: '45px',
        },
      },
    },
    MuiTablePagination: {
      root: {
        fontSize: '.6rem',
      },
      caption: {
        fontSize: '.6rem',
      },
      menuItem: {
        fontSize: '.6rem',
      },
    },
    MuiTabs: {
      root: {
        marginBottom: '1rem',
      },
    },
    MuiToolbar: {
      gutters: {
        '@media (min-width: 600px)': {
          paddingLeft: '16px',
        },
      },
    },
    RaListToolbar: {
      toolbar: {
        alignItems: 'start',
        '& form:has(> div:nth-child(3))': {
          paddingBottom: '.6rem',
        },
      },
      actions: {
        paddingRight: '0!important',
        marginTop: '-8px',
        textWrap: 'nowrap',
        '@media (max-width: 599.95px)': {
          marginTop: '3px',
        },
        '& .MuiButton-text': {
          height: '2.5rem',
          padding: '7px 10px',
          marginRight: '0',
        },
      },
    },
    RaTopToolbar: {
      root: {
        '& div:first-of-type > div:first-of-type': {
          display: 'flex',
          flexWrap: 'wrap',
          rowGap: '10px',
        },
        '& div:first-of-type > div:first-of-type button': {
          height: '2rem',
        },
        '& div:first-of-type > div:first-of-type .MuiIconButton-root': {
          padding: '0',
          marginRight: '2rem',
        },
        '& div:first-of-type > div:nth-of-type(2)': {
          height: '2rem',
        },
      },
    },
    RaToolbar: {
      toolbar: {
        backgroundColor: 'white',
      },
    },
    RaFilterButton: {
      root: {
        textWrap: 'nowrap',
        '& button': {
          '@media (max-width: 599.95px)': {
            padding: '12px',
          },
        },
        "&[resource*='song']": {
          marginLeft: '10px',
        },
      },
    },
    RaDeleteWithUndoButton: {
      deleteButton: {
        color: 'rgba(0,0,0,.57)',
        '&:hover': {
          backgroundColor: 'rgba(0, 0, 0, 0.04)',
        },
      },
    },
    RaAutocompleteSuggestionList: {
      suggestionsContainer: {
        borderRadius: '4px',
        outline: '1px solid black',
        backgroundColor: 'white',
      },
    },
    RaEmpty: {
      message: {
        marginTop: '3rem',
      },
      icon: {
        display: 'none',
      },
    },
    RaAutocompleteArrayInput: {
      chipContainerOutlined: {
        '&:empty': {
          margin: '0',
        },
        margin: '10px 0',
      },
      chip: {
        margin: '4px 4px 4px 0!important',
      },
      inputInput: {
        flexGrow: '0',
        '& #genre_id': {
          flexGrow: '0',
        },
      },
    },
    RaLayout: {
      content: {
        width: '100%',
      },
    },
    RaDatagrid: {
      headerCell: {
        fontWeight: 'bold',
      },
    },
    NDAlbumShow: {
      albumActions: {
        padding: '0',
        alignItems: 'center',
        margin: '1rem 0',
      },
    },
    MuiCardContent: {
      root: {
        fontFamily: 'monospace',
        fontSize: '.8rem',
        '& #now-playing-title': {
          fontSize: '.8rem',
        },
        '&:last-child': {
          paddingBottom: '16px',
        },
        '&[class*="makeStyles-usernameWrap-"]': {
          paddingBottom: '16px',
        },
      },
    },
    MuiDialogContent: {
      root: {
        '& .MuiTableCell-sizeSmall:last-child': {
          textAlign: 'left',
        },
      },
    },
    MuiGridList: {
      root: {
        '&:empty': {
          display: 'none',
        },
        backgroundColor: 'white',
        borderRadius: '4px',
      },
    },
    MuiGridListTile: {
      root: {
        '@media (max-width: 599.95px)': {
          padding: '7px!important',
        },
      },
      tile: {
        '& img': {
          borderRadius: '5px',
        },
      },
    },
    NDAlbumGridView: {
      root: {
        '&:has(.MuiGridList-root:empty)': {
          display: 'none',
        },
      },
      albumContainer: {
        border: '1px solid white',
        borderRadius: '5px',
        '& a:hover img': {
          outline: '1px solid black',
        },
        '& a:hover > div:nth-of-type(2)': {
          border: 'none',
          outline: '1px solid black',
        },
      },
      albumLink: {
        paddingRight: '6px',
      },
      albumSubtitle: {
        fontFamily: 'monospace',
      },
      tileBar: {
        transition: 'all 50ms ease-out',
      },
      tileBarMobile: {
        transition: 'all 50ms ease-out',
        borderLeft: '1px solid black',
        borderRight: '1px solid black',
        borderBottom: '1px solid black',
      },
    },
    MuiGridListTileBar: {
      root: {
        height: '30px!important',
        background: 'white!important',
        borderTop: '1px solid black',
        borderBottom: '1px solid black',
        borderRadius: '0 0 5px 5px',
      },
      titleWrap: {
        marginLeft: '0px',
      },
      titlePositionBottom: {
        bottom: '0',
      },
      subtitle: {
        '& button': {
          color: 'black!important',
        },
      },
      actionIcon: {
        '& button': {
          color: 'black!important',
        },
      },
    },
    RaFilter: {
      form: {
        width: '100%',
        '& div.filter-field:first-child': {
          flex: '1 100%',
          '& [class*="RaSearchInput-input-"]': {
            width: '100%',
          },
        },
      },
    },
    MuiInputAdornment: {
      positionEnd: {
        justifyContent: 'flex-end',
      },
    },
    RaFilterFormInput: {
      body: {
        '& label': {
          transform: 'translate(14px, -6px) scale(0.75)!important',
          backgroundColor: '#fafafa',
          padding: '0 5px',
        },
      },
      hideButton: {
        order: '1',
        marginLeft: '2px',
        top: '-7px',
        padding: '8px',
      },
      spacer: {
        order: '2',
      },
    },
    RaPaginationActions: {
      actions: {
        '& button': {
          border: 'none',
          fontSize: '.6rem',
        },
      },
    },
    NDAlbumDetails: {
      cover: {
        borderRadius: '5px',
      },
      content: {
        padding: '0',
        marginLeft: '1rem',
      },
      externalLinks: {
        marginTop: '5px',
      },
      notes: {
        display: 'none',
      },
      root: {
        '& p': {
          fontSize: '.7rem',
          backgroundColor: '#e0e0e0',
          borderRadius: '10px',
          width: 'fit-content',
          padding: '2px 7px',
        },
      },
    },
    NDPlaylistDetails: {
      cover: {
        borderRadius: '5px',
      },
    },
    NDDesktopArtistDetails: {
      cover: {
        borderRadius: '0px',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgb(255 255 255 / 51%), rgb(250 250 250))!important',
      },
    },
    NDArtistShow: {
      actionsContainer: {
        '& button': {
          padding: '4px 5px',
          fontSize: '0.8125rem',
          height: '2rem',
        },
      },
    },
    NDAudioPlayer: {
      audioTitle: {
        color: 'black',
      },
    },
    NDLogin: {
      main: {
        background: 'white',
        '& .MuiFormLabel-root': {
          color: '#000',
        },
        '& .MuiFormLabel-root.Mui-error': {
          color: '#000',
        },
        '& .MuiInput-underline:before': {
          borderBottom: 'none',
        },
        '& .MuiInput-underline:after': {
          borderBottom: 'none',
        },
        '& .MuiFormHelperText-root.Mui-error': {
          color: '#000',
          paddingLeft: '10px',
        },
        '& .MuiInput-underline:hover:not(.Mui-disabled):before': {
          borderBottom: 'none',
        },
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        backgroundColor: '#ffffffe6',
        border: '1px solid black',
      },
      avatar: {
        marginTop: '1rem',
        '& img': {
          filter: 'invert(1)',
        },
      },
      icon: {},
      input: {
        '& .MuiInput-root': {
          border: '1px solid black',
          borderRadius: '4px',
          padding: '10px',
        },
        '& .MuiInputLabel-root': {
          padding: '10px',
        },
        '& .MuiInputLabel-shrink': {
          transform: 'translate(0, -5.5px) scale(0.75)',
        },
      },
      actions: {
        marginTop: '2rem',
      },
      button: {
        boxShadow: 'none',
        '&:hover': {
          boxShadow: 'none',
          backgroundColor: 'rgb(117, 177, 44)',
        },
      },
      systemNameLink: {
        fontFamily: 'monospace',
        marginBottom: '1rem',
        color: 'black',
        '&:before': {
          content: "'Welcome to '",
        },
        '&:after': {
          content: "' *~*!'",
        },
      },
    },
    MuiCssBaseline: {
      '@global': {
        '*::-webkit-scrollbar': {
          display: 'none',
        },
      },
    },
    MuiBackdrop: {
      root: {
        backgroundColor: 'rgba(255, 255, 255, 0.5)',
      },
    },
    RaLoading: {
      message: {
        fontFamily: 'monospace',
      },
    },
  },
  player: {
    theme: 'light',
    stylesheet,
  },
}
