import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function AdminServices() {
  const api = useApi()
  const [rows, setRows] = useState([])
  const [cats, setCats] = useState([])
  const [q, setQ] = useState('')
  const [editing, setEditing] = useState(null)

  const load = () => {
    const params = new URLSearchParams({ limit: '500' })
    if (q) params.set('q', q)
    api.get('/admin/services?' + params).then((r) => setRows(r.data || [])).catch(() => {})
  }
  useEffect(() => { api.get('/categories').then((r) => setCats(r.data || [])).catch(() => {}); load() }, [])

  const add = () => setEditing({ category_id: cats[0]?.id, name: '', description: '', long_description: '', price_cents: 0, currency: 'RUB', duration_minutes: 60, is_from_price: false, is_active: true, sort_order: 0 })
  const edit = (s) => setEditing({ ...s, price_cents: s.price_cents })
  const del = async (s) => { if (confirm(`Архивировать «${s.name}»?`)) { await api.del('/admin/services/' + s.id); load() } }
  const save = async () => {
    try {
      if (editing.id) await api.put('/admin/services/' + editing.id, editing)
      else await api.post('/admin/services', editing)
      setEditing(null); load()
    } catch (e) { alert(e.message) }
  }

  return (
    <>
      <div className="admin-top">
        <h1 style={{ margin: 0 }}>Услуги</h1>
        <div className="row-between">
          <input className="field" placeholder="Поиск…" value={q} onChange={(e) => { setQ(e.target.value); load() }} />
          <button className="btn btn-primary" onClick={add}>+ Добавить</button>
        </div>
      </div>

      {editing && (
        <div className="card" style={{ marginBottom: 18 }}>
          <div className="row-between">
            <h3 style={{ margin: 0 }}>{editing.id ? 'Редактировать услугу' : 'Новая услуга'}</h3>
            <button className="btn btn-ghost sm" onClick={() => setEditing(null)}>✕</button>
          </div>
          <div className="row-between">
            <div className="field" style={{ flex: 2 }}>
              <label>Название</label>
              <input value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Категория</label>
              <select value={editing.category_id} onChange={(e) => setEditing({ ...editing, category_id: Number(e.target.value) })}>
                {cats.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
          </div>
          <div className="field"><label>Описание</label>
            <input value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} />
          </div>
          <div className="row-between">
            <div className="field" style={{ flex: 1 }}><label>Цена (копейки)</label>
              <input type="number" value={editing.price_cents} onChange={(e) => setEditing({ ...editing, price_cents: Number(e.target.value) })} />
              <div className="hint">350000 = 3 500 ₽</div>
            </div>
            <div className="field" style={{ flex: 1 }}><label>Валюта</label>
              <select value={editing.currency} onChange={(e) => setEditing({ ...editing, currency: e.target.value })}>
                <option>RUB</option><option>USD</option><option>EUR</option>
              </select>
            </div>
            <div className="field" style={{ flex: 1 }}><label>Длительность, мин</label>
              <input type="number" value={editing.duration_minutes} onChange={(e) => setEditing({ ...editing, duration_minutes: Number(e.target.value) })} />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Опции</label>
              <div className="row-between">
                <label><input type="checkbox" checked={editing.is_from_price} onChange={(e) => setEditing({ ...editing, is_from_price: e.target.checked })} /> от</label>
                <label><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> активен</label>
              </div>
            </div>
          </div>
          <button className="btn btn-primary" onClick={save}>Сохранить</button>
        </div>
      )}

      <div className="card"><div className="table-wrap">
        <table className="data">
          <thead><tr><th>Услуга</th><th>Категория</th><th>Цена</th><th>Активна</th><th></th></tr></thead>
          <tbody>
            {rows.map((s) => (
              <tr key={s.id}>
                <td>{s.name}<div className="muted" style={{ fontSize: 12 }}>{s.description}</div></td>
                <td>{s.category_name}</td>
                <td>{(s.price_cents / 100).toLocaleString('ru-RU')} {s.currency}{s.is_from_price ? ' (от)' : ''}</td>
                <td>{s.is_active ? '✓' : '—'}</td>
                <td>
                  <button className="btn sm" onClick={() => edit(s)}>✎</button>{' '}
                  <button className="btn sm" onClick={() => del(s)}>🗑</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div></div>
    </>
  )
}