import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function AdminAudit() {
  const api = useApi()
  const [rows, setRows] = useState([])
  useEffect(() => {
    api.get('/admin/audit?limit=100').then((r) => setRows(r.data || [])).catch(() => {})
  }, [])
  return (
    <>
      <div className="admin-top"><h1 style={{ margin: 0 }}>Журнал изменений</h1></div>
      <div className="card"><div className="table-wrap">
        <table className="data">
          <thead><tr><th>Когда</th><th>Действие</th><th>Сущность</th><th>ID</th><th>Diff</th></tr></thead>
          <tbody>
            {rows.map((e) => (
              <tr key={e.id}>
                <td className="muted">{e.created_at}</td>
                <td><span className="tag">{e.action}</span></td>
                <td>{e.entity}</td>
                <td>{e.entity_id}</td>
                <td className="muted" style={{ maxWidth: 500 }}>{JSON.stringify(e.diff)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div></div>
    </>
  )
}