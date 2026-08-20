import Link from 'next/link'
import { getAllArticles } from '../lib/content'
export default function AlsoRead({ current, category }){const items=getAllArticles().filter(a=>a.slug!==current && (!category || a.category===category)).slice(0,2);return <aside className="my-10 rounded-3xl border border-[var(--line)] bg-[var(--soft)] p-6"><h3 className="display text-2xl">Читайте также</h3><div className="mt-4 grid gap-3">{items.map(a=><Link key={a.slug} href={`/articles/${a.slug}`} className="font-semibold text-[var(--accent2)]">{a.title}</Link>)}</div></aside>}
