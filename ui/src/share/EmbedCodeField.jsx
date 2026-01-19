import React, { useState } from 'react'
import {
  Box,
  Typography,
  TextField,
  IconButton,
  Tabs,
  Tab,
  makeStyles,
  Snackbar,
} from '@material-ui/core'
import { useTranslate } from 'react-admin'
import FileCopyIcon from '@material-ui/icons/FileCopy'

const useStyles = makeStyles((theme) => ({
  root: {
    marginBottom: theme.spacing(3),
  },
  tabPanel: {
    marginTop: theme.spacing(2),
  },
  codeField: {
    fontFamily: 'monospace',
    fontSize: '12px',
    '& .MuiInputBase-root': {
      fontFamily: 'monospace',
      fontSize: '12px',
    },
  },
  copyButton: {
    marginLeft: theme.spacing(1),
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    marginBottom: theme.spacing(1),
  },
}))

const TabPanel = ({ children, value, index, ...other }) => {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`embed-tabpanel-${index}`}
      aria-labelledby={`embed-tab-${index}`}
      {...other}
    >
      {value === index && <Box py={2}>{children}</Box>}
    </div>
  )
}

export const EmbedCodeField = ({ url, title = 'Music Player' }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [tabValue, setTabValue] = useState(0)
  const [snackbarOpen, setSnackbarOpen] = useState(false)

  const handleTabChange = (event, newValue) => {
    setTabValue(newValue)
  }

  const handleCopy = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      setSnackbarOpen(true)
    })
  }

  const handleSnackbarClose = () => {
    setSnackbarOpen(false)
  }

  // 基础 iframe 嵌入代码
  const iframeEmbed = `<iframe src="${url}" width="100%" height="450" frameborder="0" allowfullscreen></iframe>`

  // 响应式 iframe 嵌入代码
  const responsiveEmbed = `<div style="position: relative; padding-bottom: 56.25%; height: 0; overflow: hidden;">
  <iframe src="${url}" style="position: absolute; top: 0; left: 0; width: 100%; height: 100%;" frameborder="0" allowfullscreen></iframe>
</div>`

  // 左下角悬浮播放器嵌入代码
  const floatingPlayerEmbed = `<!-- Navidrome 悬浮播放器 -->
<div id="navidrome-floating-player">
  <div id="nav-player-container" class="nav-collapsed">
    <div id="nav-player-toggle" onclick="toggleNavPlayer()">
      <span id="nav-toggle-icon">🎵</span>
    </div>
    <div id="nav-player-content">
      <iframe src="${url}" frameborder="0" allowfullscreen></iframe>
    </div>
  </div>
</div>

<style>
#navidrome-floating-player {
  position: fixed;
  left: 20px;
  bottom: 20px;
  z-index: 9999;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}

#nav-player-container {
  background: white;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  transition: all 0.3s ease;
}

#nav-player-container.nav-collapsed {
  width: 60px;
  height: 60px;
}

#nav-player-container.nav-expanded {
  width: 380px;
  height: 520px;
}

#nav-player-toggle {
  width: 60px;
  height: 60px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 12px;
  transition: all 0.3s ease;
}

#nav-player-toggle:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.4);
}

#nav-toggle-icon {
  font-size: 28px;
  transition: transform 0.3s ease;
}

#nav-player-container.nav-expanded #nav-toggle-icon {
  transform: rotate(90deg);
}

#nav-player-content {
  display: none;
  width: 380px;
  height: 460px;
}

#nav-player-container.nav-expanded #nav-player-content {
  display: block;
}

#nav-player-content iframe {
  width: 100%;
  height: 100%;
  border: none;
}

/* 移动端适配 */
@media (max-width: 768px) {
  #navidrome-floating-player {
    left: 10px;
    bottom: 10px;
  }

  #nav-player-container.nav-expanded {
    width: calc(100vw - 20px);
    height: 480px;
    max-width: 380px;
  }

  #nav-player-content {
    width: 100%;
  }
}
</style>

<script>
function toggleNavPlayer() {
  const container = document.getElementById('nav-player-container');
  if (container.classList.contains('nav-collapsed')) {
    container.classList.remove('nav-collapsed');
    container.classList.add('nav-expanded');
  } else {
    container.classList.remove('nav-expanded');
    container.classList.add('nav-collapsed');
  }
}

// 可选：点击播放器外部区域时收起
document.addEventListener('click', function(event) {
  const player = document.getElementById('navidrome-floating-player');
  const container = document.getElementById('nav-player-container');

  if (player && !player.contains(event.target) &&
      container.classList.contains('nav-expanded')) {
    toggleNavPlayer();
  }
});
</script>`

  // 右下角悬浮播放器（备选）
  const floatingPlayerRightEmbed = floatingPlayerEmbed
    .replace('left: 20px;', 'right: 20px;')
    .replace('left: 10px;', 'right: 10px;')

  const embedOptions = [
    {
      label: translate('message.floatingPlayerLeft'),
      code: floatingPlayerEmbed,
      description: translate('message.floatingPlayerLeftDesc'),
    },
    {
      label: translate('message.basicIframe'),
      code: iframeEmbed,
      description: translate('message.basicIframeDesc'),
    },
    {
      label: translate('message.responsiveIframe'),
      code: responsiveEmbed,
      description: translate('message.responsiveIframeDesc'),
    },
    {
      label: translate('message.floatingPlayerRight'),
      code: floatingPlayerRightEmbed,
      description: translate('message.floatingPlayerRightDesc'),
    },
  ]

  return (
    <Box className={classes.root}>
      <Typography variant="body2" color="textSecondary" gutterBottom>
        {translate('message.embedCode')}
      </Typography>

      <Tabs
        value={tabValue}
        onChange={handleTabChange}
        indicatorColor="primary"
        textColor="primary"
        variant="scrollable"
        scrollButtons="auto"
      >
        {embedOptions.map((option, index) => (
          <Tab key={index} label={option.label} />
        ))}
      </Tabs>

      {embedOptions.map((option, index) => (
        <TabPanel key={index} value={tabValue} index={index}>
          <Typography
            variant="caption"
            color="textSecondary"
            display="block"
            gutterBottom
          >
            {option.description}
          </Typography>

          <Box display="flex" alignItems="flex-start">
            <TextField
              fullWidth
              multiline
              rows={option.code.split('\n').length > 20 ? 20 : 12}
              variant="outlined"
              value={option.code}
              className={classes.codeField}
              InputProps={{
                readOnly: true,
              }}
            />
            <IconButton
              className={classes.copyButton}
              onClick={() => handleCopy(option.code)}
              color="primary"
              size="small"
              title={translate('message.copyCode')}
            >
              <FileCopyIcon />
            </IconButton>
          </Box>

          <Typography variant="caption" color="textSecondary" display="block">
            {translate('message.embedTip')}
          </Typography>
        </TabPanel>
      ))}

      <Snackbar
        open={snackbarOpen}
        autoHideDuration={2000}
        onClose={handleSnackbarClose}
        message={translate('message.codeCopied')}
      />
    </Box>
  )
}
