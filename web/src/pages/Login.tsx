import { useState } from 'react'
import { login } from '../lib/api.ts'

interface Props {
  onLogin: () => void
}

export default function Login({ onLogin }: Props) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(password)
      onLogin()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-apple-bg px-4">
      <div className="w-full max-w-sm bg-white rounded-2xl shadow-sm border border-apple-border p-8">
        <h1 className="text-xl font-semibold text-center mb-6">Signal Sideband</h1>
        <form onSubmit={handleSubmit}>
          <input
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Password"
            autoFocus
            className="w-full px-4 py-2.5 rounded-xl border border-apple-border bg-white text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 focus:border-apple-blue transition-colors"
          />
          {error && (
            <p className="mt-2 text-sm text-red-500">{error}</p>
          )}
          <button
            type="submit"
            disabled={loading || !password}
            className="mt-4 w-full px-5 py-2.5 bg-apple-blue text-white text-sm font-medium rounded-xl hover:bg-apple-blue-hover transition-colors disabled:opacity-50"
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}
