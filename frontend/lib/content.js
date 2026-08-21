import fs from 'node:fs'
import path from 'node:path'
import matter from 'gray-matter'

export const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://men40.an51.su'
export const categories = [
  { slug: 'health-energy', name: 'Здоровье и энергия', description: 'Сон, восстановление, профилактические чекапы и бережная энергия без диагнозов.' },
  { slug: 'style', name: 'Стиль', description: 'Гардероб, уход и визуальная собранность без суеты и подростковых трендов.' },
  { slug: 'body-form', name: 'Форма и тело', description: 'Тренировки 40+, осанка и подвижность с уважением к реальному графику.' },
  { slug: 'relationships-confidence', name: 'Отношения и уверенность', description: 'Карьера, семья и личная опора без клише о кризисе среднего возраста.' },
]
const dir = path.join(process.cwd(), 'content/articles')
export function getAllArticles(){
  return fs.readdirSync(dir).filter(f=>f.endsWith('.md')).map(file=>{
    const raw = fs.readFileSync(path.join(dir,file),'utf8')
    const { data, content } = matter(raw)
    return { slug:file.replace(/\.md$/,''), content, readingTime: Math.max(4, Math.round(content.split(/\s+/).length/180)), ...data }
  }).sort((a,b)=>new Date(b.date)-new Date(a.date))
}
export function getArticle(slug){return getAllArticles().find(a=>a.slug===slug)}
export function getCategory(slug){return categories.find(c=>c.slug===slug)}
export function getArticlesByCategory(slug){return getAllArticles().filter(a=>a.category===slug)}
export function absolute(pathname=''){return new URL(pathname, siteUrl).toString()}
export function articleJsonLd(article) {
  return {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: article.description,
    url: absolute(`/articles/${article.slug}`),
    inLanguage: 'ru-RU',
    articleSection: getCategory(article.category)?.name,
    datePublished: article.date,
    dateModified: article.date,
    image: absolute(article.coverImage),
    author: { '@type': 'Organization', name: 'Редакция «На высоте»' },
    publisher: {
      '@type': 'Organization',
      name: 'На высоте',
      logo: { '@type': 'ImageObject', url: absolute('/favicon.svg') },
    },
    mainEntityOfPage: absolute(`/articles/${article.slug}`),
  }
}
export function breadcrumbs(items){return {'@context':'https://schema.org','@type':'BreadcrumbList', itemListElement:items.map((it,i)=>({'@type':'ListItem', position:i+1, name:it.name, item:absolute(it.href)}))}}
export function collectionJsonLd(category, articles){return {'@context':'https://schema.org','@type':'CollectionPage', name:category?.name || 'Все статьи', description:category?.description || 'Журнал На высоте', url:absolute(category?`/category/${category.slug}`:'/'), inLanguage:'ru-RU', hasPart:articles.map(a=>({'@type':'Article', headline:a.title, description:a.description, image:absolute(a.coverImage), url:absolute(`/articles/${a.slug}`)}))}}
export function renderMarkdown(source, currentSlug, category){
  const escape = (s)=>s.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;')
  const inline = (s)=>escape(s).replace(/\[([^\]]+)\]\(([^)]+)\)/g,'<a href="$2">$1</a>')
  const related = getAllArticles().filter(a=>a.slug!==currentSlug && (!category || a.category===category)).slice(0,2)
  return source.split(/\r?\n[ \t]*\r?\n/).map(block=>{
    const b=block.trim()
    if(!b) return ''
    if(b.startsWith('Читайте также:')) return `<aside class="my-10 rounded-3xl border border-[var(--line)] bg-[var(--soft)] p-6"><h3 class="display text-2xl">Читайте также</h3><p>${inline(b.replace('Читайте также:','').trim())}</p><div class="mt-4 grid gap-3">${related.map(a=>`<a class="font-semibold text-[var(--accent2)]" href="/articles/${a.slug}">${escape(a.title)}</a>`).join('')}</div></aside>`
    if(b.startsWith('## ')) return `<h2>${inline(b.slice(3))}</h2>`
    if(b.startsWith('> ')) return `<blockquote>${inline(b.slice(2))}</blockquote>`
    return `<p>${inline(b)}</p>`
  }).join('\n')
}
