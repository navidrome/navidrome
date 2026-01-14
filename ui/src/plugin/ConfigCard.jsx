import React, { useCallback } from 'react'
import {
  Card,
  CardContent,
  Typography,
  TextField as MuiTextField,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Paper,
} from '@material-ui/core'
import { MdDelete } from 'react-icons/md'

export const ConfigCard = ({
  configPairs,
  onConfigPairsChange,
  classes,
  translate,
}) => {
  const handleKeyChange = useCallback(
    (index, newKey) => {
      const newPairs = [...configPairs]
      newPairs[index] = { ...newPairs[index], key: newKey }
      onConfigPairsChange(newPairs)
    },
    [configPairs, onConfigPairsChange],
  )

  const handleValueChange = useCallback(
    (index, newValue) => {
      const newPairs = [...configPairs]
      newPairs[index] = { ...newPairs[index], value: newValue }
      onConfigPairsChange(newPairs)
    },
    [configPairs, onConfigPairsChange],
  )

  const handleDeleteRow = useCallback(
    (index) => {
      const newPairs = configPairs.filter((_, i) => i !== index)
      onConfigPairsChange(newPairs)
    },
    [configPairs, onConfigPairsChange],
  )

  const handleAddRow = useCallback(() => {
    onConfigPairsChange([...configPairs, { key: '', value: '' }])
  }, [configPairs, onConfigPairsChange])

  return (
    <Card className={classes.section}>
      <CardContent>
        <Typography variant="h6" className={classes.sectionTitle}>
          {translate('resources.plugin.sections.configuration')}
        </Typography>
        <Typography variant="body2" color="textSecondary" gutterBottom>
          {translate('resources.plugin.messages.configHelp')}
        </Typography>

        <TableContainer component={Paper} variant="outlined">
          <Table size="small" className={classes.configTable}>
            <TableHead>
              <TableRow>
                <TableCell width="40%">
                  {translate('resources.plugin.fields.configKey')}
                </TableCell>
                <TableCell width="50%">
                  {translate('resources.plugin.fields.configValue')}
                </TableCell>
                <TableCell width="10%" align="right">
                  <IconButton
                    size="small"
                    onClick={handleAddRow}
                    aria-label={translate('resources.plugin.actions.addConfig')}
                    className={classes.configActionIconButton}
                  >
                    +
                  </IconButton>
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {configPairs.map((pair, index) => (
                <TableRow key={index}>
                  <TableCell>
                    <MuiTextField
                      fullWidth
                      size="small"
                      variant="outlined"
                      value={pair.key}
                      onChange={(e) => handleKeyChange(index, e.target.value)}
                      placeholder={translate(
                        'resources.plugin.placeholders.configKey',
                      )}
                      InputProps={{
                        className: classes.configTableInput,
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    <MuiTextField
                      fullWidth
                      size="small"
                      variant="outlined"
                      multiline
                      minRows={1}
                      value={pair.value}
                      onChange={(e) => handleValueChange(index, e.target.value)}
                      placeholder={translate(
                        'resources.plugin.placeholders.configValue',
                      )}
                      InputProps={{
                        className: classes.configTableInput,
                      }}
                      inputProps={{
                        style: { resize: 'vertical' },
                      }}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      onClick={() => handleDeleteRow(index)}
                      aria-label={translate('ra.action.delete')}
                      className={classes.configActionIconButton}
                    >
                      <MdDelete />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
              {configPairs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} align="center">
                    <Typography variant="body2" color="textSecondary">
                      {translate('resources.plugin.messages.noConfig')}
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </CardContent>
    </Card>
  )
}
