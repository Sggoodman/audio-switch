import React, { useEffect, useState, useCallback } from 'react'
import { createTheme, ThemeProvider } from '@mui/material/styles'
import {
  Box,
  Typography,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Button,
  Alert,
  CircularProgress,
  Chip,
  Card,
  CardContent,
  Divider
} from '@mui/material'
import SpeakerIcon from '@mui/icons-material/Speaker'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import KeyboardIcon from '@mui/icons-material/Keyboard'
import SwapHorizIcon from '@mui/icons-material/SwapHoriz'
import ComputerIcon from '@mui/icons-material/Computer'
import VolumeUpIcon from '@mui/icons-material/VolumeUp'

const THEME_DIC = {
  light: createTheme({
    palette: {
      mode: 'light',
      primary: { main: '#6366F1' },
      secondary: { main: '#8B5CF6' }
    },
    typography: { fontFamily: 'system-ui' }
  }),
  dark: createTheme({
    palette: {
      mode: 'dark',
      primary: { main: '#818CF8' },
      secondary: { main: '#A78BFA' }
    },
    typography: { fontFamily: 'system-ui' }
  })
}

const platformLabels = {
  win32: 'Windows',
  darwin: 'macOS',
  linux: 'Linux'
}

export default function App() {
  const [theme, setTheme] = useState(window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
  const [loading, setLoading] = useState(true)
  const [devices, setDevices] = useState([])
  const [error, setError] = useState(null)
  const [switching, setSwitching] = useState(false)
  const [switchResult, setSwitchResult] = useState(null)
  const [platform, setPlatform] = useState(null)
  const [requirements, setRequirements] = useState(null)

  const loadDevices = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await window.services.getAudioDevices()
      if (result.error) {
        if (result.error === 'AudioDeviceCmdlets') {
          setError({ type: 'requirements', ...result })
        } else {
          setError({ type: 'failed', message: result.message })
        }
        setDevices([])
      } else {
        setDevices(Array.isArray(result) ? result : [])
      }
    } catch (e) {
      setError({ type: 'failed', message: e.message })
      setDevices([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    window.utools.onPluginEnter(({ code, type, payload, from }) => {
      // 路由分发
    })

    window.utools.onPluginOut(() => {
      // 清理
    })

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
      setTheme(e.matches ? 'dark' : 'light')
    })

    // 初始化
    const init = async () => {
      const p = await window.services.getPlatform()
      setPlatform(p)

      const req = await window.services.checkRequirements()
      setRequirements(req)

      if (req.ready) {
        await loadDevices()
      } else {
        setLoading(false)
      }
    }
    init()
  }, [loadDevices])

  const handleSwitch = async () => {
    try {
      setSwitching(true)
      setSwitchResult(null)
      const result = await window.services.switchAudioDevice()
      setSwitchResult(result)
      if (result.success) {
        await loadDevices()
      }
    } catch (e) {
      setSwitchResult({ success: false, message: e.message })
    } finally {
      setSwitching(false)
    }
  }

  const handleSetupHotkey = () => {
    window.services.redirectHotKeySetting()
  }

  const handleRefresh = () => {
    loadDevices()
  }

  return (
    <ThemeProvider theme={THEME_DIC[theme]}>
      <Box sx={{
        width: '100%',
        height: '100vh',
        boxSizing: 'border-box',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.default',
        color: 'text.primary'
      }}>
        {/* Header */}
        <Box sx={{
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'center',
          gap: 1.5
        }}>
          <VolumeUpIcon color="primary" />
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            音频输出切换
          </Typography>
          {platform && (
            <Chip label={platformLabels[platform] || platform} size="small" />
          )}
        </Box>

        {/* Content */}
        <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
          {/* 需求检查失败提示 */}
          {requirements && !requirements.ready && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                {requirements.message}
              </Typography>
              {requirements.hint && (
                <Typography variant="body2" sx={{ mt: 0.5, fontFamily: 'monospace' }}>
                  {requirements.hint}
                </Typography>
              )}
            </Alert>
          )}

          {/* 错误提示 */}
          {error && error.type === 'failed' && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error.message}
            </Alert>
          )}

          {/* 操作结果 */}
          {switchResult && (
            <Alert
              severity={switchResult.success ? 'success' : 'error'}
              sx={{ mb: 2 }}
              onClose={() => setSwitchResult(null)}
            >
              {switchResult.success
                ? `已切换到: ${switchResult.deviceName}`
                : `切换失败: ${switchResult.message}`}
            </Alert>
          )}

          {/* 加载状态 */}
          {loading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          )}

          {/* 设备列表 */}
          {!loading && requirements?.ready && (
            <Card sx={{ mb: 2 }}>
              <CardContent sx={{ pb: 1 }}>
                <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                  音频输出设备
                </Typography>
              </CardContent>
              <List sx={{ pt: 0 }}>
                {devices.length === 0 && (
                  <ListItem>
                    <ListItemText primary="未检测到音频输出设备" />
                  </ListItem>
                )}
                {devices.map((device, index) => (
                  <ListItem key={device.id || index} sx={{ py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 36 }}>
                      {device.isDefault ? (
                        <CheckCircleIcon color="primary" fontSize="small" />
                      ) : (
                        <SpeakerIcon color="action" fontSize="small" />
                      )}
                    </ListItemIcon>
                    <ListItemText
                      primary={device.name}
                      secondary={device.isDefault ? '当前使用' : ''}
                      primaryTypographyProps={{
                        variant: 'body2',
                        fontWeight: device.isDefault ? 600 : 400
                      }}
                    />
                  </ListItem>
                ))}
              </List>
            </Card>
          )}

          {/* 说明 */}
          {!loading && requirements?.ready && devices.length >= 2 && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              点击「切换音频」可将输出切换到下一个设备。您也可以设置全局快捷键来快速切换。
            </Typography>
          )}
        </Box>

        {/* Footer */}
        <Box sx={{
          p: 2,
          borderTop: 1,
          borderColor: 'divider',
          display: 'flex',
          gap: 1,
          flexWrap: 'wrap'
        }}>
          <Button
            variant="contained"
            size="small"
            startIcon={<SwapHorizIcon />}
            onClick={handleSwitch}
            disabled={loading || switching || !requirements?.ready || devices.length < 2}
          >
            {switching ? '切换中...' : '切换音频'}
          </Button>
          <Button
            variant="outlined"
            size="small"
            startIcon={<KeyboardIcon />}
            onClick={handleSetupHotkey}
          >
            设置快捷键
          </Button>
          <Button
            variant="text"
            size="small"
            onClick={handleRefresh}
            disabled={loading}
          >
            刷新
          </Button>
        </Box>
      </Box>
    </ThemeProvider>
  )
}