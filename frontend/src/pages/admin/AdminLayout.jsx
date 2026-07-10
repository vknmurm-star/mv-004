import { Link, NavLink, Outlet, Navigate, useNavigate } from 'react-router-dom'
import { useAuth } from '../../utils/auth.jsx'

const NAV = [
  { to: '/admin', label: 'Дашборд', end: true },
  { to: '/admin/services', label: 'Услуги' },
  { to: '/admin/prices/import', label: 'Импорт прайса' },
  { to: '/admin/appointments', label: 'Заявки' },
  { to: '/admin/leads', label: 'Лиды (MAX)' },
  { to: '/admin/faq', label: 'FAQ' },
  { to: '/admin/knowledge', label: 'База знаний AI' },
  { to: '/admin/reviews', label: 'Отзывы' },
  { to: '/admin/audit', label: 'Журнал' },
]

export default function AdminLayout() {
  const { token, clear } = useAuth()
  const nav = useNavigate()
  if (!token) return <Navigate to="/admin/login" replace />

  const logout = () => { clear(); nav('/admin/login') }

  return (
    <div className="admin-shell">
      <aside className="admin-side">
        <Link to="/" className="brand" style={{ margin: '8px 12px 16px' }}>
          <span className="brand-mark">🕒</span><span className="brand-name">Управление</span>
        </Link>
        {NAV.map((n) => (
          <NavLink key={n.to} to={n.to} end={n.end}
            className={({ isActive }) => isActive ? 'active' : ''}>{n.label}</NavLink>
        ))}
        <button className="btn btn-ghost sm" style={{ marginTop: 'auto' }} onClick={logout}>Выйти</button>
      </aside>
      <div className="admin-main">
        <Outlet />
      </div>
    </div>
  )
}