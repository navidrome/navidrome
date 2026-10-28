import { withStyles } from '@mui/material/styles'
import MuiDialogContent from '@mui/material/DialogContent'

export const DialogContent = // TODO(v6): migrate withStyles → styled()
withStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
  },
}))(MuiDialogContent)
