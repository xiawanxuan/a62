import { ref, onUnmounted } from 'vue'
import type { WSMessage, Annotation, OnlineUser, Point } from '@/types'

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const isConnected = ref(false)
  const onlineUsers = ref<OnlineUser[]>([])
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5

  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let fileId: string | null = null
  let userId: string | null = null

  const onAnnotationCreate = ref<((ann: Annotation) => void) | null>(null)
  const onAnnotationUpdate = ref<((ann: Annotation) => void) | null>(null)
  const onAnnotationDelete = ref<((id: string) => void) | null>(null)
  const onCursorMove = ref<((userId: string, point: Point) => void) | null>(null)
  const onUserJoin = ref<((user: OnlineUser) => void) | null>(null)
  const onUserLeave = ref<((userId: string) => void) | null>(null)

  const connect = (fileIdParam: string, userIdParam: string, userName: string) => {
    fileId = fileIdParam
    userId = userIdParam

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/annotate/${fileId}?userId=${userId}&userName=${encodeURIComponent(userName)}`

    try {
      ws.value = new WebSocket(wsUrl)

      ws.value.onopen = () => {
        isConnected.value = true
        reconnectAttempts.value = 0
        sendMessage('user-join', { name: userName })
      }

      ws.value.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)
          handleMessage(message)
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      ws.value.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      ws.value.onclose = () => {
        isConnected.value = false
        scheduleReconnect()
      }
    } catch (e) {
      console.error('Failed to create WebSocket:', e)
      scheduleReconnect()
    }
  }

  const scheduleReconnect = () => {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      console.error('Max reconnect attempts reached')
      return
    }

    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 10000)
    reconnectTimer = setTimeout(() => {
      reconnectAttempts.value++
      if (fileId && userId) {
        connect(fileId, userId, 'User')
      }
    }, delay)
  }

  const handleMessage = (message: WSMessage) => {
    switch (message.type) {
      case 'annotation-create':
        if (onAnnotationCreate.value) {
          onAnnotationCreate.value(message.payload as Annotation)
        }
        break
      case 'annotation-update':
        if (onAnnotationUpdate.value) {
          onAnnotationUpdate.value(message.payload as Annotation)
        }
        break
      case 'annotation-delete':
        if (onAnnotationDelete.value) {
          onAnnotationDelete.value(message.payload.id)
        }
        break
      case 'cursor-move':
        if (onCursorMove.value) {
          onCursorMove.value(message.userId, message.payload as Point)
        }
        break
      case 'user-join':
        const newUser: OnlineUser = {
          id: message.userId,
          name: message.payload.name,
          color: message.payload.color || generateColor(message.userId),
          cursor: null
        }
        onlineUsers.value.push(newUser)
        if (onUserJoin.value) {
          onUserJoin.value(newUser)
        }
        break
      case 'user-leave':
        onlineUsers.value = onlineUsers.value.filter(u => u.id !== message.userId)
        if (onUserLeave.value) {
          onUserLeave.value(message.userId)
        }
        break
    }
  }

  const sendMessage = (type: string, payload: any) => {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) return

    const message: WSMessage = {
      type,
      payload,
      userId: userId || '',
      timestamp: Date.now()
    }

    ws.value.send(JSON.stringify(message))
  }

  const sendAnnotationCreate = (annotation: Annotation) => {
    sendMessage('annotation-create', annotation)
  }

  const sendAnnotationUpdate = (annotation: Annotation) => {
    sendMessage('annotation-update', annotation)
  }

  const sendAnnotationDelete = (id: string) => {
    sendMessage('annotation-delete', { id })
  }

  const sendCursorMove = (point: Point) => {
    sendMessage('cursor-move', point)
  }

  const disconnect = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    if (ws.value) {
      sendMessage('user-leave', {})
      ws.value.close()
      ws.value = null
    }

    isConnected.value = false
    onlineUsers.value = []
  }

  const generateColor = (id: string): string => {
    let hash = 0
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash)
    }
    const hue = Math.abs(hash % 360)
    return `hsl(${hue}, 70%, 50%)`
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    isConnected,
    onlineUsers,
    connect,
    disconnect,
    sendAnnotationCreate,
    sendAnnotationUpdate,
    sendAnnotationDelete,
    sendCursorMove,
    onAnnotationCreate,
    onAnnotationUpdate,
    onAnnotationDelete,
    onCursorMove,
    onUserJoin,
    onUserLeave
  }
}
