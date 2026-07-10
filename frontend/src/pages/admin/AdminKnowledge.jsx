import { useEffect, useState } from 'react'
import { useApi } from '../../utils/api.js'

export default function AdminKnowledge() {
  const api = useApi()
  const [items, setItems] = useState([])
  const [editing, setEditing] = useState(null)

  const load = () => api.get('/admin/knowledge').then((r) => setItems(r.data || [])).catch(() => {})
  useEffect(load, [])

  const save = async () => {
    try {
      const body = { ...editing, tags: (editing.tagsRaw || '').split(',').map((t) => t.trim()).filter(Boolean) }
      const upd = await api.put('/admin/knowledge/' + (editing.id || 0), body)
      setEditing(null); load()
    } catch (e) { alert(e.message) }
  }
  const del = async (id) => { if (confirm('Удалить?')) { await api.del('/admin/knowledge/' + id); load() } }

  return (
    <>
      <div className="admin-top">
        <div>
          <h1 style={{ margin: 0 }}>База знаний AI</h1>
          <div className="muted">Эти записи использует ИИ-ассистент в MAX для ответов. Пишите кратко и по делу.</div>
        </div>
        <button className="btn btn-primary" onClick={() => setEditing({ title: '', body: '', tagsRaw: '', source: '', id: 0 })}>+ Добавить</button>
      </div>

      {editing && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div className="row-between"><h3 style={{ margin: 0 }}>{editing.id ? 'Редактировать' : 'Новая запись'}</h3>
            <button className="btn btn-ghost sm" onClick={() => setEditing(null)}>✕</button></div>
          <div className="field"><label>Заголовок</label><input value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} /></div>
          <div className="field"><label>Текст (ответ/факт)</label><textarea value={editing.body} onChange={(e) => setEditing({ ...editing, body: e.target.value })} /></div>
          <div className="field"><label>Теги (через запятую)</label><input value={editing.tagsRaw} onChange={(e) => setEditing({ ...editing, tagsRaw: e.target.value })} /></div>
          <div className="field"><label>Источник</label><input value={editing.source} onChange={(e) => setEditing({ ...editing, source: e.target.value })} /></div>
          <button className="btn btn-primary" onClick={save}>Сохранить</button>
        </div>
      )}

      <div className="grid">
        {items.map((k) => (
          <div className="card" key={k.id}>
            <div className="row-between"><h3 style={{ margin: 0 }}>{k.title}</h3>
              <div><button className="btn sm" onClick={() => setEditing({ ...k, tagsRaw: (k.tags || []).join(', ') })}>✎</button>{' '}
                <button className="btn sm" onClick={() => del(k.id)}>🗑</button></div></div>
            <p className="muted">{k.body}</p>
            <div className="tag-row-tags">{(k.tags || []).map((t) => <span className="tag" key={t}>{t}</span>)}</div>
          </div>
        ))}
      </div>
    </>
  )
}