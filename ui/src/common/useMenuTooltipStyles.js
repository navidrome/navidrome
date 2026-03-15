import { makeStyles } from '@material-ui/core/styles'

const useMenuTooltipStyles = makeStyles(
  (theme) => ({
    tooltip: {
      backgroundColor:
        theme.palette.type === 'dark'
          ? 'rgba(97, 97, 97, 0.92)'
          : 'rgba(224, 224, 224, 0.92)',
      color:
        theme.palette.type === 'dark'
          ? theme.palette.common.white
          : theme.palette.common.black,
      borderRadius: theme.shape.borderRadius,
      ...theme.typography.body2,
      padding: theme.spacing(0.5, 1),
      maxWidth: 300,
    },
  }),
  { name: 'NDOverflowTooltip' },
)

export default useMenuTooltipStyles
