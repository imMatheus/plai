import { LiveGameBoard } from './LiveGameBoard'
import { GamesList } from './GamesList'

function App() {
  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#f9fafb',
      fontFamily: 'system-ui, sans-serif'
    }}>
      <div style={{
        backgroundColor: '#fff',
        borderBottom: '1px solid #e5e7eb',
        padding: '1rem 0'
      }}>
        <h1 style={{
          fontSize: '2rem',
          fontWeight: 'bold',
          textAlign: 'center',
          margin: 0
        }}>
          AI Chess Battle: Claude vs ChatGPT
        </h1>
      </div>

      <LiveGameBoard />

      <div style={{
        borderTop: '2px solid #e5e7eb',
        marginTop: '2rem',
        paddingTop: '2rem'
      }}>
        <GamesList />
      </div>
    </div>
  )
}

export default App
