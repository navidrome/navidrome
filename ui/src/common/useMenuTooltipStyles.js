import { makeStyles } from '@material-ui/core/styles'

const useMenuTooltipStyles = makeStyles((theme) => {
  const paperRoot = theme.overrides?.MuiPaper?.root || {}

  return {
    tooltip: {
      backgroundColor:
        paperRoot.backgroundColor || theme.palette.background.paper,
      color: paperRoot.color || theme.palette.text.primary,
      boxShadow: theme.shadows[8],
      borderRadius: theme.shape.borderRadius,
      ...theme.typography.body1,
      padding: theme.spacing(1, 2),
      maxWidth: '30vw',
    },
  }
})

export default useMenuTooltipStyles
