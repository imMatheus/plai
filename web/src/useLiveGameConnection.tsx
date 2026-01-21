import { useEffect, useRef, useState } from "react"

type GameMessage = {
    type: "game_state" | "move"
    fen: string
    move?: string
    turn: "white" | "black"
    svg: string
}

export function useLiveGameConnection() {
    const [fen, setFen] = useState<string>("")
    const [lastMove, setLastMove] = useState<string | null>(null)
    const [turn, setTurn] = useState<"white" | "black">("white")
    const [svg, setSvg] = useState<string>("")
    const [isConnected, setIsConnected] = useState(false)
    const ws = useRef<WebSocket | null>(null)

    useEffect(() => {
        // Connect to WebSocket
        ws.current = new WebSocket('ws://localhost:8080/ws')

        ws.current.onopen = () => {
            console.log('Connected to WebSocket')
            setIsConnected(true)
        }

        ws.current.onmessage = (event) => {
            const message: GameMessage = JSON.parse(event.data)

            if (message.type === 'game_state' || message.type === 'move') {
                setFen(message.fen)
                setTurn(message.turn)
                setSvg(message.svg)

                if (message.type === 'move' && message.move) {
                    setLastMove(message.move)
                }
            }
        }

        ws.current.onerror = (error) => {
            console.error('WebSocket error:', error)
        }

        ws.current.onclose = () => {
            console.log('Disconnected from WebSocket')
            setIsConnected(false)
        }

        // Cleanup on unmount
        return () => {
            ws.current?.close()
        }
    }, [])

    return { fen, lastMove, turn, svg, isConnected }
}