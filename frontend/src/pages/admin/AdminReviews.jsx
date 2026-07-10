import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function AdminReviews() {
  const api = useApi()
  const [items, setItems] = useState([])
  const load = () => api.get('/admin/reviews').then((r) => setItems(r.data || [])).catch(() => {})
  useEffect(load, [])
  const toggle = async (id, pub) => { await api.patch('/admin/reviews/' + id, { is_published: pub }); load() }
  return (
    <>
      <div className="admin-top"><h1 style={{ margin: 0 }}>Отзывы (модерация)</h1></div>
      <div className="grid">
        {items.map((r) => (
          <div className="card" key={r.id}>
            <div className="row-between"><strong>{r.author}</strong><span style={{ color: 'var(--accent-2)' }}>{'★'.repeat(r.rating)}</span></div>
            {r.car_info && <div className="muted" style={{ fontSize: 13 }}>{r.car_info}</div>}
            <p>{r.body}</p>
            <div className="row-between">
              <span className="tag">{r.is_published ? 'опубликован' : 'скрыт'}</span>
              <button className="btn sm" onClick={() => toggle(r.id, !r.is_published)}>
                {r.is_published ? 'Скрыть' : 'Опубликовать'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </>
  )
}