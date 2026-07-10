import { useApi } from '../utils/api.js'
import { useEffect, useState } from 'react'

export default function Reviews() {
  const api = useApi()
  const [items, setItems] = useState([])
  useEffect(() => {
    api.get('/reviews').then((r) => setItems(r.data || [])).catch(() => {})
  }, [])
  if (!items.length) return null
  return (
    <>
      <h3 style={{ margin: '24px 0 12px' }}>Отзывы клиентов</h3>
      <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(280px,1fr))' }}>
        {items.map((r) => (
          <div className="card" key={r.id}>
            <div className="row-between">
              <strong>{r.author}</strong>
              <span style={{ color: 'var(--accent-2)' }}>{'★'.repeat(r.rating)}</span>
            </div>
            {r.car_info && <div className="muted" style={{ fontSize: 13 }}>{r.car_info}</div>}
            <p style={{ margin: '10px 0 0' }}>{r.body}</p>
          </div>
        ))}
      </div>
    </>
  )
}