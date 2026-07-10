import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

const STATUSES = ['new', 'confirmed', 'done', 'canceled', 'no_show']

export default function AdminAppointments() {
  const api = useApi()
  const [rows, setRows] = useState([])
  const [status, setStatus] = useState('')
  const [editing, setEditing] = useState(null)

  const load = () => {
    const p = new URLSearchParams({ limit: '200' })
    if (status) p.set('status', status)
    api.get('/admin/appointments?' + p).then((r) => setRows(r.data || [])).catch(() => {})
  }
  useEffect(load, [status])

  const patch = async (id, body) => {
    await api.patch('/admin/appointments/' + id, body); setEditing(null); load()
  }

  return (
    <>
      <div className="admin-top">
        <h1 style={{ margin: 0 }}>Заявки</h1>
        <div className="tabs">
          {['', ...STATUSES].map((s) => (
            <button key={s} className={status === s ? 'active' : ''} onClick={() => setStatus(s)}>
              {s || 'Все'}
            </button>
          ))}
        </div>
      </div>
      <div className="card"><div className="table-wrap">
        <table className="data">
          <thead><tr><th>#</th><th>Клиент</th><th>Авто</th><th>Услуга</th><th>Жел. дата</th><th>Статус</th><th></th></tr></thead>
          <tbody>
            {rows.map((a) => (
              <tr key={a.id}>
                <td>{a.id}</td>
                <td>{a.name}<div className="muted" style={{ fontSize: 12 }}>{a.phone}</div></td>
                <td>{a.car_make} {a.car_model}<div className="muted" style={{ fontSize: 12 }}>{a.gov_number}</div></td>
                <td>{a.service_id || '—'}</td>
                <td className="muted">{a.desired_at?.slice(0, 16).replace('T', ' ') || '—'}</td>
                <td><span className="tag">{a.status}</span></td>
                <td><button className="btn sm" onClick={() => setEditing(a)}>✎</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div></div>

      {editing && (
        <div className="card" style={{ position: 'fixed', right: 24, bottom: 24, width: 360, boxShadow: 'var(--shadow)' }}>
          <div className="row-between"><h3 style={{ margin: 0 }}>Заявка #{editing.id}</h3>
            <button className="btn btn-ghost sm" onClick={() => setEditing(null)}>✕</button></div>
          <p className="muted" style={{ fontSize: 13 }}>{editing.name} · {editing.phone}<br />{editing.car_make} {editing.car_model}</p>
          <div className="field"><label>Статус</label>
            <select value={editing.status} onChange={(e) => setEditing({ ...editing, status: e.target.value })}>
              {STATUSES.map((s) => <option key={s}>{s}</option>)}
            </select>
          </div>
          <div className="field"><label>Заметка менеджера</label>
            <textarea value={editing.manager_note} onChange={(e) => setEditing({ ...editing, manager_note: e.target.value })} />
          </div>
          <button className="btn btn-primary" onClick={() => patch(editing.id, { status: editing.status, manager_note: editing.manager_note })}>Сохранить</button>
        </div>
      )}
    </>
  )
}