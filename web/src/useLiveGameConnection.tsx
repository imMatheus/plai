import { useEffect, useRef, useState } from "react"

type GameMessage = {
    type: "game_state" | "move"
    fen: string
    move?: string
    turn: "white" | "black"
    svg: string
    viewerCount: number
    whitePlayer: string
    blackPlayer: string
    currentPlayer?: string
}

export function useLiveGameConnection() {
    const [fen, setFen] = useState<string>("")
    const [lastMove, setLastMove] = useState<string | null>(null)
    const [turn, setTurn] = useState<"white" | "black">("white")
    const [svg, setSvg] = useState<string>("")
    const [isConnected, setIsConnected] = useState(false)
    const [viewerCount, setViewerCount] = useState<number>(0)
    const [whitePlayer, setWhitePlayer] = useState<string>("")
    const [blackPlayer, setBlackPlayer] = useState<string>("")
    const [currentPlayer, setCurrentPlayer] = useState<string>("")
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
            console.log('Received message:', message.type, {
                fen: message.fen,
                svgLength: message.svg?.length,
                whitePlayer: message.whitePlayer,
                blackPlayer: message.blackPlayer
            })

            if (message.type === 'game_state' || message.type === 'move') {
                setFen(message.fen)
                setTurn(message.turn)
                setSvg(message.svg)
                setViewerCount(message.viewerCount)
                setWhitePlayer(message.whitePlayer)
                setBlackPlayer(message.blackPlayer)

                if (message.currentPlayer) {
                    setCurrentPlayer(message.currentPlayer)
                }

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

    return { fen, lastMove, turn, svg, isConnected, viewerCount, whitePlayer, blackPlayer, currentPlayer }
}