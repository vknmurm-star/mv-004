import { useState } from 'react'
import { useNavigate, Navigate } from 'react-router-dom'
import { useApi } from '../../utils/api.js'
import { useAuth } from '../../utils/auth.jsx'

export default function AdminLogin() {
  const api = useApi()
  const { token, setToken } = useAuth()
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState('')

  if (token) return <Navigate to="/admin" replace />

  const submit = async (e) => {
    e.preventDefault()
    setErr('')
    try {
      const r = await api.post('/admin/login', { email, password })
      setToken(r.data.token)
      nav('/admin')
    } catch (e2) {
      setErr(e2.message)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', background: 'var(--bg)' }}>
      <form className="card" style={{ width: 360 }} onSubmit={submit}>
        <h2 style={{ marginTop: 0 }}>Вход в админку</h2>
        {err && <div className="error-banner">{err}</div>}
        <div className="field">
          <label>Email</label>
          <input value={email} onChange={(e) => setEmail(e.target.value)} placeholder="admin@timemachine.ru" />
        </div>
        <div className="field">
          <label>Пароль</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="admin12345" />
        </div>
        <button className="btn btn-primary" style={{ width: '100%' }}>Войти</button>
        <div className="hint" style={{ marginTop: 10 }}>Demo: admin@timemachine.ru / admin12345</div>
      </form>
    </div>
  )
}