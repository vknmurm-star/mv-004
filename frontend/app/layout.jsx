import './globals.css'
import Link from 'next/link'
import { Fraunces } from 'next/font/google'
import { categories } from '../lib/content'

const fraunces = Fraunces({ subsets: ['latin', 'cyrillic'], variable: '--font-display', display: 'swap' })
const siteUrl = process.env.NEXT_PUBLIC_SITE_URL || 'https://men40.an51.su'

export const metadata = {
  metadataBase: new URL(siteUrl),
  title: { default: 'На высоте — журнал для мужчин 40+', template: '%s | На высоте' },
  description: 'Спокойный журнал о здоровье, стиле, форме и уверенности для мужчин 40+.',
  alternates: { canonical: '/' },
  openGraph: {
    type: 'website',
    locale: 'ru_RU',
    siteName: 'На высоте',
    title: 'На высоте — журнал для мужчин 40+',
    description: 'Спокойный журнал о здоровье, стиле, форме и уверенности для мужчин 40+.',
    url: '/',
    images: [{ url: '/images/articles/men-001.jpg', width: 1200, height: 630, alt: 'Мужчина смотрит на панораму города на закате со смотровой площадки' }],
  },
  twitter: {
    card: 'summary_large_image',
    title: 'На высоте — журнал для мужчин 40+',
    description: 'Спокойный журнал о здоровье, стиле, форме и уверенности для мужчин 40+.',
    images: ['/images/articles/men-001.jpg'],
  },
  verification: {
    google: 'jYc47eoIqDRwS-9o-00BHj3gmiPZbrRKDXE05FXM8m4',
    yandex: 'YANDEX_WEBMASTER_VERIFICATION_CODE',
  },
  robots: { index: true, follow: true },
}

function Logo(){return <Link href="/" className="flex items-center gap-3"><span className="grid h-11 w-11 place-items-center rounded-full border border-[var(--accent)] bg-[var(--ink)] text-[var(--accent)] font-bold">НВ</span><span><b className="display block text-xl leading-none">На высоте</b><small className="text-[var(--muted)]">мужчинам 40+</small></span></Link>}

export default function RootLayout({ children }) {
  return <html lang="ru" className={fraunces.variable}><body><header className="sticky top-0 z-20 border-b border-[var(--line)] bg-[rgba(244,239,231,.86)] backdrop-blur"><div className="container flex items-center justify-between py-4"><Logo/><nav className="hidden gap-5 text-sm font-semibold text-[var(--muted)] md:flex">{categories.map(c=><Link key={c.slug} href={`/category/${c.slug}`}>{c.name}</Link>)}</nav></div></header>{children}<footer className="mt-20 border-t border-[var(--line)] py-10"><div className="container grid gap-6 md:grid-cols-[1fr_2fr]"><Logo/><div><p className="text-[var(--muted)]">Контент носит информационный характер и не заменяет консультацию профильного специалиста.</p><p className="mt-4 text-sm text-[var(--muted)]">© {new Date().getFullYear()} На высоте. Варианты названия: «Вторая опора», «Курс 40+», «Собранный возраст».</p></div></div></footer></body></html>
}
