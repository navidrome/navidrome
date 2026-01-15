import { makeStyles } from '@material-ui/core/styles'

export const usePluginShowStyles = makeStyles(
  (theme) => ({
    root: {
      padding: theme.spacing(2),
      maxWidth: 900,
    },
    section: {
      marginBottom: theme.spacing(3),
    },
    sectionTitle: {
      marginBottom: theme.spacing(1),
      fontWeight: 600,
    },
    manifestBox: {
      backgroundColor:
        theme.palette.type === 'dark'
          ? theme.palette.grey[900]
          : theme.palette.grey[100],
      padding: theme.spacing(2),
      borderRadius: theme.shape.borderRadius,
      fontFamily: 'monospace',
      fontSize: '0.85rem',
      whiteSpace: 'pre-wrap',
      wordBreak: 'break-word',
      overflow: 'auto',
      maxHeight: 400,
    },
    saveButton: {
      marginTop: theme.spacing(2),
    },
    infoGrid: {
      '& .MuiGrid-item': {
        paddingTop: theme.spacing(0.5),
        paddingBottom: theme.spacing(0.5),
      },
    },
    infoLabel: {
      fontWeight: 500,
      color: theme.palette.text.secondary,
    },
    pathField: {
      fontFamily: 'monospace',
      fontSize: '0.85rem',
      wordBreak: 'break-all',
    },
    permissionsContainer: {
      display: 'flex',
      flexWrap: 'wrap',
      gap: theme.spacing(0.5),
    },
    permissionChip: {
      fontSize: '0.75rem',
    },
    tooltipContent: {
      '& code': {
        fontFamily: 'monospace',
        fontSize: '0.8em',
        backgroundColor: 'rgba(255,255,255,0.1)',
        padding: '1px 4px',
        borderRadius: 2,
      },
    },
    configTable: {
      '& .MuiTableCell-root': {
        padding: theme.spacing(1),
      },
    },
    configTableInput: {
      fontFamily: 'monospace',
      fontSize: '0.85rem',
    },
    configActionIconButton: {
      backgroundColor: theme.palette.action.hover,
      borderRadius: theme.shape.borderRadius,
      padding: theme.spacing(0.5, 1),
      fontWeight: 700,
      '&:hover': {
        backgroundColor: theme.palette.action.selected,
      },
    },
  }),
  { name: 'NDPluginShow' },
)
