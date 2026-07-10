import { useEffect, useMemo, useState } from 'react'
import { useApi } from '../utils/api.js'
import { formatPrice, formatDuration } from '../utils/format.js'

export default function Prices() {
  const api = useApi()
  const [rows, setRows] = useState([])
  const [cats, setCats] = useState([])
  const [q, setQ] = useState('')
  const [cat, setCat] = useState('')

  useEffect(() => {
    api.get('/categories').then((r) => setCats(r.data || [])).catch(() => {})
    load()
  }, [cat, q])

  const load = () => {
    const params = new URLSearchParams({ limit: '500' })
    if (cat) params.set('category', cat)
    if (q) params.set('q', q)
    api.get('/services?' + params).then((r) => setRows(r.data || [])).catch(() => {})
  }

  const groups = useMemo(() => {
    const m = new Map()
    rows.forEach((r) => {
      if (!m.has(r.category_name)) m.set(r.category_name, [])
      m.get(r.category_name).push(r)
    })
    return [...m.entries()]
  }, [rows])

  return (
    <section className="section container">
      <div className="row-between">
        <div>
          <h1 style={{ margin: 0 }}>Прайс-лист</h1>
          <div className="lead">Цены открытые. Признак «от» означает предварительную стоимость — точную смету формируем после осмотра.</div>
        </div>
        <button className="btn" onClick={() => api.download('/prices/export', 'timemachine-prices.csv')}>⬇ Экспорт CSV</button>
      </div>

      <div className="row-between" style={{ margin: '18px 0' }}>
        <input className="field" style={{ maxWidth: 320 }} placeholder="Поиск…" value={q} onChange={(e) => setQ(e.target.value)} />
        <select className="field" style={{ maxWidth: 280 }} value={cat} onChange={(e) => setCat(e.target.value)}>
          <option value="">Все категории</option>
          {cats.map((c) => <option key={c.id} value={c.slug}>{c.name}</option>)}
        </select>
      </div>

      {groups.map(([name, list]) => (
        <div className="card" key={name} style={{ marginBottom: 18 }}>
          <h3 style={{ margin: '0 0 12px' }}>{name}</h3>
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr><th>Услуга</th><th>Описание</th><th>Длительность</th><th style={{ textAlign: 'right' }}>Цена</th></tr>
              </thead>
              <tbody>
                {list.map((s) => (
                  <tr key={s.id}>
                    <td>{s.name}</td>
                    <td className="muted">{s.description}</td>
                    <td className="muted">{formatDuration(s.duration_minutes)}</td>
                    <td style={{ textAlign: 'right' }}>
                      <strong>{formatPrice(s.price_cents, s.currency, s.is_from_price)}</strong>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ))}
    </section>
  )
}