import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function Dashboard() {
  const api = useApi()
  const [stats, setStats] = useState({ services: 0, appointments: 0, leads: 0, faq: 0 })

  useEffect(() => {
    Promise.all([
      api.get('/admin/services?limit=1'),
      api.get('/admin/appointments?limit=1'),
      api.get('/admin/leads?limit=1'),
      api.get('/admin/faq')
    ]).then(([s, a, l, f]) => {
      setStats({
        services: s.meta?.total ?? 0,
        appointments: a.meta?.total ?? 0,
        leads: l.meta?.total ?? 0,
        faq: (f.data || []).length
      })
    }).catch(() => {})
  }, [])

  return (
    <>
      <div className="admin-top"><h1 style={{ margin: 0 }}>Дашборд</h1></div>
      <div className="stat-grid">
        {[
          { num: stats.services, lbl: 'Услуг в прайсе' },
          { num: stats.appointments, lbl: 'Заявок с сайта' },
          { num: stats.leads, lbl: 'Лидов из MAX' },
          { num: stats.faq, lbl: 'Ответов в FAQ' },
        ].map((s) => (
          <div className="card stat" key={s.lbl}>
            <div className="num">{s.num}</div>
            <div className="lbl">{s.lbl}</div>
          </div>
        ))}
      </div>
      <div className="card" style={{ marginTop: 18 }}>
        <h3>Быстрые действия</h3>
        <div className="row-between">
          <a className="btn" href="#/admin/prices/import">Импорт прайса из CSV/XLSX</a>
          <a className="btn" href="#/admin/appointments">Новые заявки</a>
          <a className="btn" href="#/admin/leads">Лиды из MAX</a>
        </div>
      </div>
    </>
  )
}