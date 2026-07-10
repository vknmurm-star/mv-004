import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function AdminFaq() {
  const api = useApi()
  const [items, setItems] = useState([])
  const [editing, setEditing] = useState(null)

  const load = () => api.get('/admin/faq').then((r) => setItems(r.data || [])).catch(() => {})
  useEffect(load, [])

  const save = async () => {
    try {
      if (editing.id) await api.put('/admin/faq/' + editing.id, editing)
      else await api.post('/admin/faq', editing)
      setEditing(null); load()
    } catch (e) { alert(e.message) }
  }
  const del = async (id) => { if (confirm('Удалить?')) { await api.del('/admin/faq/' + id); load() } }

  return (
    <>
      <div className="admin-top"><h1 style={{ margin: 0 }}>FAQ</h1>
        <button className="btn btn-primary" onClick={() => setEditing({ question: '', answer: '', category: '', sort_order: 0, is_published: true })}>+ Добавить</button>
      </div>
      {editing && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div className="row-between"><h3 style={{ margin: 0 }}>{editing.id ? 'Редактировать' : 'Новый FAQ'}</h3>
            <button className="btn btn-ghost sm" onClick={() => setEditing(null)}>✕</button></div>
          <div className="field"><label>Вопрос</label><input value={editing.question} onChange={(e) => setEditing({ ...editing, question: e.target.value })} /></div>
          <div className="field"><label>Ответ</label><textarea value={editing.answer} onChange={(e) => setEditing({ ...editing, answer: e.target.value })} /></div>
          <div className="row-between">
            <div className="field" style={{ flex: 1 }}><label>Категория</label><input value={editing.category} onChange={(e) => setEditing({ ...editing, category: e.target.value })} /></div>
            <div className="field" style={{ flex: 1 }}><label>Порядок</label><input type="number" value={editing.sort_order} onChange={(e) => setEditing({ ...editing, sort_order: Number(e.target.value) })} /></div>
            <div className="field" style={{ flex: 1 }}>
              <label>Опубликован</label><label><input type="checkbox" checked={editing.is_published} onChange={(e) => setEditing({ ...editing, is_published: e.target.checked })} /> Да</label>
            </div>
          </div>
          <button className="btn btn-primary" onClick={save}>Сохранить</button>
        </div>
      )}
      <div className="card"><div className="table-wrap">
        <table className="data">
          <thead><tr><th>Вопрос</th><th>Категория</th><th>Опубл.</th><th></th><th></th></tr></thead>
          <tbody>
            {items.map((f) => (
              <tr key={f.id}>
                <td>{f.question}<div className="muted" style={{ fontSize: 12 }}>{f.answer}</div></td>
                <td>{f.category || '—'}</td>
                <td>{f.is_published ? '✓' : '—'}</td>
                <td><button className="btn sm" onClick={() => setEditing(f)}>✎</button></td>
                <td><button className="btn sm" onClick={() => del(f.id)}>🗑</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div></div>
    </>
  )
}