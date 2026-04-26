import { useState, useEffect, useRef, useCallback } from 'preact/hooks'

interface UseWebSocketOptions<T = unknown> {
  onMessage?: (data: T) => void
  onError?: (error: Event) => void
  onOpen?: () => void
  onClose?: () => void
  reconnectInterval?: number
  reconnectAttempts?: number
}

interface UseWebSocketReturn<T = unknown> {
  isConnected: boolean
  lastMessage: T | null
  send: (data: unknown) => void
  disconnect: () => void
}

export function useWebSocket<T = unknown>(
  url: string,
  options: UseWebSocketOptions<T> = {}
): UseWebSocketReturn<T> {
  const {
    reconnectInterval = 3000,
    reconnectAttempts = 5,
  } = options

  // Store callbacks in refs to avoid dependency instability
  const onMessageRef = useRef(options.onMessage)
  onMessageRef.current = options.onMessage
  const onErrorRef = useRef(options.onError)
  onErrorRef.current = options.onError
  const onOpenRef = useRef(options.onOpen)
  onOpenRef.current = options.onOpen
  const onCloseRef = useRef(options.onClose)
  onCloseRef.current = options.onClose

  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<T | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectCountRef = useRef(0)
  const reconnectTimeoutRef = useRef<number>()
  const intentionalCloseRef = useRef(false)

  const connect = useCallback(() => {
    try {
      const ws = new WebSocket(url)

      ws.onopen = () => {
        setIsConnected(true)
        reconnectCountRef.current = 0
        onOpenRef.current?.()
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as T
          setLastMessage(data)
          onMessageRef.current?.(data)
        } catch (error) {
          if (import.meta.env.DEV) console.error('Failed to parse WebSocket message:', error)
        }
      }

      ws.onerror = (error) => {
        if (import.meta.env.DEV) console.error('WebSocket error:', error)
        onErrorRef.current?.(error)
      }

      ws.onclose = () => {
        setIsConnected(false)
        onCloseRef.current?.()

        // Auto-reconnect only if not intentionally closed
        if (!intentionalCloseRef.current && reconnectCountRef.current < reconnectAttempts) {
          reconnectCountRef.current++
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }

      wsRef.current = ws
    } catch (error) {
      if (import.meta.env.DEV) console.error('Failed to create WebSocket:', error)
    }
  }, [url, reconnectInterval, reconnectAttempts])

  useEffect(() => {
    intentionalCloseRef.current = false
    connect()

    return () => {
      intentionalCloseRef.current = true
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  const send = useCallback((data: unknown) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  const disconnect = useCallback(() => {
    intentionalCloseRef.current = true
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
    }
  }, [])

  return { isConnected, lastMessage, send, disconnect }
}
