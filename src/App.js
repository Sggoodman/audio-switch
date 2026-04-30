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
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Slider,
  Grid
} from '@mui/material'
import SpeakerIcon from '@mui/icons-material/Speaker'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import KeyboardIcon from '@mui/icons-material/Keyboard'
import InfoIcon from '@mui/icons-material/Info'
import VolumeUpIcon from '@mui/icons-material/VolumeUp'
import SwapHorizIcon from '@mui/icons-material/SwapHoriz'
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
  const [platform, setPlatform] = useState(null)
  const [requirements, setRequirements] = useState(null)
  const [switching, setSwitching] = useState(false)
  const [switchResult, setSwitchResult] = useState(null)
  const [selectedDevice1, setSelectedDevice1] = useState('')
  const [selectedDevice2, setSelectedDevice2] = useState('')
  const [volume1, setVolume1] = useState(50)
  const [volume2, setVolume2] = useState(50)
  const [saveStatus, setSaveStatus] = useState(null)

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
    window.utools.onPluginEnter(({ code }) => {
      if (code === 'audio-quick-switch') {
        window.services.switchAudioDevice().then(result => {
          if (result.success) {
            window.utools.showNotification(`已切换到: ${result.deviceName}`)
          } else {
            window.utools.showNotification(`切换失败: ${result.message}`)
          }
          window.utools.outPlugin()
        })
      }
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
        // 加载偏好设备配置
        const preferred = window.services.getPreferredDevices()
        if (preferred) {
          setSelectedDevice1(preferred.device1?.id || '')
          setSelectedDevice2(preferred.device2?.id || '')
          setVolume1(preferred.device1?.volume ?? 50)
          setVolume2(preferred.device2?.volume ?? 50)
        }
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

  const handleSavePreferred = () => {
    if (!selectedDevice1 || !selectedDevice2 || selectedDevice1 === selectedDevice2) return
    const dev1 = devices.find(d => d.id === selectedDevice1)
    const dev2 = devices.find(d => d.id === selectedDevice2)
    if (!dev1 || !dev2) return
    const result = window.services.savePreferredDevices(
      { id: dev1.id, name: dev1.name, volume: volume1 },
      { id: dev2.id, name: dev2.name, volume: volume2 }
    )
    if (result.success) {
      setSaveStatus({ type: 'success', message: '配置已保存' })
    } else {
      setSaveStatus({ type: 'error', message: result.message })
    }
    setTimeout(() => setSaveStatus(null), 2000)
  }

  const handleTestVolume = async (deviceId, volume) => {
    const result = await window.services.setDeviceVolume(deviceId, volume)
    if (result.success) {
      window.utools.showNotification('音量已设置')
      await loadDevices()
    } else {
      window.utools.showNotification(`设置失败: ${result.message}`)
    }
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

          {/* 切换结果 */}
          {switchResult && (
            <Alert
              severity={switchResult.success ? 'success' : 'error'}
              sx={{ mb: 2, mt: 1 }}
              onClose={() => setSwitchResult(null)}
            >
              {switchResult.success
                ? `已切换到: ${switchResult.deviceName}`
                : `切换失败: ${switchResult.message}`}
            </Alert>
          )}

          {/* 偏好设备选择 */}
          {!loading && requirements?.ready && devices.length >= 2 && (
            <Card sx={{ mb: 2, mt: 1 }}>
              <CardContent>
                <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                  快捷切换设置
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
                  选择两个常用设备，快捷键将在它们之间互相切换。若当前设备不在此列表中，将切换到设备 A。
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <FormControl size="small" fullWidth sx={{ mb: 1 }}>
                      <InputLabel>设备 A</InputLabel>
                      <Select
                        value={selectedDevice1}
                        label="设备 A"
                        onChange={e => setSelectedDevice1(e.target.value)}
                      >
                        {devices.map(d => (
                          <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <Typography variant="caption" color="text.secondary">
                      预设音量: {volume1}%
                    </Typography>
                    <Slider
                      value={volume1}
                      onChange={e => setVolume1(e.target.value)}
                      min={0}
                      max={100}
                      size="small"
                      sx={{ mt: 0.5 }}
                    />
                  </Grid>
                  <Grid item xs={6}>
                    <FormControl size="small" fullWidth sx={{ mb: 1 }}>
                      <InputLabel>设备 B</InputLabel>
                      <Select
                        value={selectedDevice2}
                        label="设备 B"
                        onChange={e => setSelectedDevice2(e.target.value)}
                      >
                        {devices.map(d => (
                          <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <Typography variant="caption" color="text.secondary">
                      预设音量: {volume2}%
                    </Typography>
                    <Slider
                      value={volume2}
                      onChange={e => setVolume2(e.target.value)}
                      min={0}
                      max={100}
                      size="small"
                      sx={{ mt: 0.5 }}
                    />
                  </Grid>
                </Grid>
                <Box sx={{ display: 'flex', gap: 1, mt: 1.5 }}>
                  <Button
                    variant="contained"
                    size="small"
                    onClick={handleSavePreferred}
                    disabled={!selectedDevice1 || !selectedDevice2 || selectedDevice1 === selectedDevice2}
                  >
                    保存配置
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    onClick={() => selectedDevice1 && handleTestVolume(selectedDevice1, volume1)}
                    disabled={!selectedDevice1}
                  >
                    测试设备 A 音量
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    onClick={() => selectedDevice2 && handleTestVolume(selectedDevice2, volume2)}
                    disabled={!selectedDevice2}
                  >
                    测试设备 B 音量
                  </Button>
                </Box>
                {saveStatus && (
                  <Alert
                    severity={saveStatus.type}
                    sx={{ mt: 1, py: 0 }}
                  >
                    {saveStatus.message}
                  </Alert>
                )}
              </CardContent>
            </Card>
          )}

          {/* 使用说明 */}
          {!loading && requirements?.ready && (
            <Alert severity="info" sx={{ mb: 2 }} icon={<InfoIcon />}>
              <Typography variant="body2">
                在 uTools 输入「快捷切换音频」可一键切换设备。建议为该指令绑定全局快捷键。
              </Typography>
            </Alert>
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
            {switching ? '切换中...' : '立即切换'}
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