import React from 'react'
import Alert from '@mui/material/Alert'
import AlertTitle from '@mui/material/AlertTitle'
import Button from '@mui/material/Button'

const GLOBAL_ERROR_ID = 'global-error-' + Date.now()

function hashString(str) {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return 'error-' + (hash >>> 0);
}

function showGlobalError(err, type) {
  if (!err.name || !err.message) return
  const errorId = hashString(err.name + ':' + err.message)
  let cardEl = document.getElementById(errorId)
  if (cardEl) {
    let countEl = cardEl.querySelector('.global-error-count')
    if (!countEl) {
      countEl = document.createElement('span')
      countEl.className = 'global-error-count'
      countEl.textContent = '2'
      cardEl.querySelector('.global-error-title')?.prepend(countEl)
    } else {
      countEl.textContent = String(parseInt(countEl.textContent) + 1)
    }
    return
  }
  let containerEl = document.getElementById(GLOBAL_ERROR_ID)
  if (!containerEl) {
    containerEl = document.createElement('div')
    containerEl.id = GLOBAL_ERROR_ID
    containerEl.className = 'global-error'
    document.body.appendChild(containerEl)
  }
  const title = (type === 'global' ? '未捕获的错误': (type === 'promise' ? '未处理的异步错误' : ''))
  const stack = err.stack?.replace(/file:\/\/\/.*?([^\/\\]+:\d+:\d+)/g, '$1') || err.message
  cardEl = document.createElement('div')
  cardEl.id = errorId
  cardEl.className = 'global-error-card'
  const titleEl = document.createElement('div')
  titleEl.className = 'global-error-title'
  titleEl.textContent = title
  const stackEl = document.createElement('pre')
  stackEl.className = 'global-error-stack'
  stackEl.textContent = stack
  const closeBtnEl = document.createElement('button')
  closeBtnEl.className = 'global-error-btn'
  closeBtnEl.textContent = '关闭'
  closeBtnEl.onclick = () => { if (cardEl && cardEl.parentNode) cardEl.parentNode.removeChild(cardEl) }
  const copyBtnEl = document.createElement('button')
  copyBtnEl.className = 'global-error-btn'
  copyBtnEl.textContent = '复制'
  copyBtnEl.onclick = () => { window.utools.copyText(title + '\n' + stack) }
  const headerEl = document.createElement('div')
  headerEl.className = 'global-error-header'
  const actionsEl = document.createElement('div')
  actionsEl.className = 'global-error-actions'
  actionsEl.appendChild(copyBtnEl)
  actionsEl.appendChild(closeBtnEl)
  headerEl.appendChild(titleEl)
  headerEl.appendChild(actionsEl)
  cardEl.appendChild(headerEl)
  if (stackEl) cardEl.appendChild(stackEl)
  containerEl.appendChild(cardEl)
}

export default class ErrorBoundary extends React.Component {
  state = {
    error: null
  }

  static getDerivedStateFromError(error) {
    error.stack = error.stack.replace(/file:\/\/\/.*?([^\/\\]+:\d+:\d+)/g, '$1')
    return { error }
  }

  componentDidMount () {
    window.addEventListener('error', event => {
      if (event.error) showGlobalError(event.error, 'global')
    })
    window.addEventListener('unhandledrejection', event => {
      if (event.reason) showGlobalError(event.reason, 'promise')
    })
  }

  handleCopyError = () => {
    const error = this.state.error
    if (!error) return
    window.utools.copyText('React 渲染错误\n' + error.stack)
  }

  render() {
    if (this.state.error) {
      return (
        <div className='render-error-alert'>
          <Alert variant='filled' severity='error' action={<Button onClick={this.handleCopyError} color='inherit'>复制</Button>}>
            <AlertTitle>React 渲染错误</AlertTitle>
            <pre>{this.state.error.stack}</pre>
          </Alert>
        </div>
      )
    }
    return this.props.children
  }
}
