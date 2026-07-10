import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

const STATUSES = ['raw', 'qualified', 'handled', 'trash']

export default function AdminLeads() {
  const api = useApi()
  const [rows, setRows] = useState([])
  const [status, setStatus] = useState('')

  const load = () => {
    const p = new URLSearchParams({ limit: '200' })
    if (status) p.set('status', status)
    api.get('/admin/leads?' + p).then((r) => setRows(r.data || [])).catch(() => {})
  }
  useEffect(load, [status])

  const patch = async (id, next) => { await api.patch('/admin/leads/' + id, { status: next }); load() }

  return (
    <>
      <div className="admin-top">
        <h1 style={{ margin: 0 }}>Лиды</h1>
        <div className="tabs">
          {['', ...STATUSES].map((s) => (
            <button key={s} className={status === s ? 'active' : ''} onClick={() => setStatus(s)}>{s || 'Все'}</button>
          ))}
        </div>
      </div>
      <div className="card"><div className="table-wrap">
        <table className="data">
          <thead><tr><th>#</th><th>Канал</th><th>Имя</th><th>Телефон</th><th>Содержание</th><th>Статус</th><th>→</th></tr></thead>
          <tbody>
            {rows.map((l) => (
              <tr key={l.id}>
                <td>{l.id}</td>
                <td>{l.channel === 'max' ? '💬 MAX' : '🌐 сайт'}</td>
                <td>{l.name || '—'}</td>
                <td className="muted">{l.phone || l.max_chat_id || '—'}</td>
                <td className="muted" style={{ maxWidth: 360 }}>{JSON.stringify(l.answer)}</td>
                <td><span className="tag">{l.status}</span></td>
                <td>
                  <select value={l.status} onChange={(e) => patch(l.id, e.target.value)}>
                    {STATUSES.map((s) => <option key={s}>{s}</option>)}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div></div>
    </>
  )
}