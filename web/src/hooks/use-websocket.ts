import { useEffect, useRef, useState, useCallback } from 'react'
import { useAuthStore } from '@/stores/auth'

export interface UseWebSocketOptions {
  /** URL path (e.g., '/api/v1/ws/scan'). Protocol and host are derived from window.location. */
  url: string
  /** Called when a parsed JSON message is received. */
  onMessage: (data: unknown) => void
  /** Called on WebSocket errors. */
  onError?: (event: Event) => void
  /** Milliseconds between reconnect attempts. Default: 3000. */
  reconnectInterval?: number
  /** Max reconnect attempts before giving up. Default: 5. */
  maxReconnectAttempts?: number
  /** Whether the WebSocket should be active. Default: true. */
  enabled?: boolean
}

export interface UseWebSocketReturn {
  isConnected: boolean
  reconnectCount: number
  disconnect: () => void
}

export function useWebSocket({
  url,
  onMessage,
  onError,
  reconnectInterval = 3000,
  maxReconnectAttempts = 5,
  enabled = true,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [isConnected, setIsConnected] = useState(false)
  const reconnectCountRef = useRef(0)
  const [reconnectCount, setReconnectCount] = useState(0)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | undefined>(undefined)
  const mountedRef = useRef(true)
  const onMessageRef = useRef(onMessage)
  const onErrorRef = useRef(onError)
  const connectRef = useRef<(() => void) | undefined>(undefined)

  // Keep callback refs current without causing reconnects.
  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    onErrorRef.current = onError
  }, [onError])

  // Update connect implementation when dependencies change.
  useEffect(() => {
    connectRef.current = () => {
      const { accessToken } = useAuthStore.getState()
      if (!accessToken || !enabled) return

      // Close existing connection.
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }

      // Build WebSocket URL from current location.
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${window.location.host}${url}?token=${accessToken}`

      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        if (!mountedRef.current) return
        setIsConnected(true)
        reconnectCountRef.current = 0
        setReconnectCount(0)
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data as string)
          onMessageRef.current(data)
        } catch {
          // Ignore non-JSON messages.
        }
      }

      ws.onerror = (event) => {
        onErrorRef.current?.(event)
      }

      ws.onclose = () => {
        if (!mountedRef.current) return
        setIsConnected(false)
        wsRef.current = null

        // Attempt reconnection.
        if (reconnectCountRef.current < maxReconnectAttempts && enabled) {
          reconnectTimerRef.current = window.setTimeout(() => {
            if (!mountedRef.current) return
            reconnectCountRef.current++
            setReconnectCount(reconnectCountRef.current)
            connectRef.current?.()
          }, reconnectInterval)
        }
      }

      wsRef.current = ws
    }
  }, [url, enabled, reconnectInterval, maxReconnectAttempts])

  // Connect/disconnect based on enabled state.
  useEffect(() => {
    mountedRef.current = true
    if (enabled) {
      connectRef.current?.()
    }

    return () => {
      mountedRef.current = false
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [enabled, url, reconnectInterval, maxReconnectAttempts])

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }
    reconnectCountRef.current = maxReconnectAttempts // Prevent reconnection.
    if (wsRef.current) {
      wsRef.current.onclose = null
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
  }, [maxReconnectAttempts])

  return { isConnected, reconnectCount, disconnect }
}
