import { Link } from 'react-router-dom'
import { formatPrice, formatDuration } from '../utils/format.js'

export default function ServicesGrid({ services = [], loading }) {
  if (loading) {
    return <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(280px,1fr))' }}>
      {[0, 1, 2, 3].map((i) => <div className="card" key={i} style={{ height: 180, opacity: .5 }} />)}
    </div>
  }
  return (
    <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(280px,1fr))' }}>
      {services.map((s) => (
        <div className="card" key={s.id} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div className="row-between">
            <span className="pill">{s.category_name}</span>
            {s.is_from_price && <span className="badge-from">от</span>}
          </div>
          <h3 style={{ margin: 0 }}>{s.name}</h3>
          {s.description && <div className="muted" style={{ fontSize: 14 }}>{s.description}</div>}
          <div className="row-between" style={{ marginTop: 'auto' }}>
            <strong style={{ fontSize: 20 }}>{formatPrice(s.price_cents, s.currency, s.is_from_price)}</strong>
            <span className="muted">{formatDuration(s.duration_minutes)}</span>
          </div>
          <Link to="/booking" className="btn btn-primary sm">Записаться</Link>
        </div>
      ))}
    </div>
  )
}