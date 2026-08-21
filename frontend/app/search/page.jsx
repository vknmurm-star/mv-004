import Link from 'next/link'
import ArticleCard from '../../components/ArticleCard'
import JsonLd from '../../components/JsonLd'
import { breadcrumbs, getAllArticles } from '../../lib/content'

export const metadata = {
  title: 'Поиск',
  description: 'Поиск материалов журнала «На высоте».',
  alternates: { canonical: '/search' },
  robots: { index: false, follow: true },
}

export default async function SearchPage({ searchParams }) {
  const params = await searchParams
  const query = (params?.q || '').trim()
  const normalized = query.toLocaleLowerCase('ru-RU')
  const results = normalized
    ? getAllArticles().filter(article => `${article.title} ${article.description} ${article.content}`.toLocaleLowerCase('ru-RU').includes(normalized))
    : []

  return <main className="container py-14">
    <JsonLd data={breadcrumbs([{ name: 'Главная', href: '/' }, { name: 'Поиск', href: '/search' }])} />
    <Link href="/" className="text-sm text-[var(--accent2)]">На главную</Link>
    <h1 className="display mt-5 text-5xl">Поиск</h1>
    <form className="mt-6 flex max-w-2xl gap-3">
      <input name="q" defaultValue={query} placeholder="Например, сон или тренировки" className="min-w-0 flex-1 rounded-full border border-[var(--line)] bg-white px-5 py-3" />
      <button className="rounded-full bg-[var(--ink)] px-6 py-3 font-semibold text-white" type="submit">Найти</button>
    </form>
    {query && <p className="mt-8 text-[var(--muted)]">Найдено материалов: {results.length}</p>}
    <div className="mt-6 grid gap-6 md:grid-cols-2">{results.map(article => <ArticleCard key={article.slug} article={article} />)}</div>
  </main>
}
