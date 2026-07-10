import { useEffect, useMemo, useState } from 'react'
import { useApi } from '../utils/api.js'
import ServicesGrid from '../components/ServicesGrid.jsx'

export default function Services() {
  const api = useApi()
  const [services, setServices] = useState([])
  const [cats, setCats] = useState([])
  const [cat, setCat] = useState('')
  const [q, setQ] = useState('')

  useEffect(() => {
    api.get('/categories').then((r) => setCats(r.data || [])).catch(() => {})
    load()
  }, [cat, q])

  const load = () => {
    const params = new URLSearchParams({ limit: '500' })
    if (cat) params.set('category', cat)
    if (q) params.set('q', q)
    api.get('/services?' + params).then((r) => setServices(r.data || [])).catch(() => {})
  }

  return (
    <section className="section container">
      <h1>Услуги</h1>
      <div className="lead">Категории работ автомастерской «Машина времени». Все услуги — на гарантии 6 месяцев.</div>

      <div className="row-between" style={{ marginBottom: 16 }}>
        <input
          className="field" style={{ maxWidth: 320 }}
          placeholder="Поиск услуги…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select className="field" style={{ maxWidth: 280 }} value={cat} onChange={(e) => setCat(e.target.value)}>
          <option value="">Все категории</option>
          {cats.map((c) => <option key={c.id} value={c.slug}>{c.name}</option>)}
        </select>
      </div>

      <ServicesGrid services={services} loading={!services.length} />
    </section>
  )
}