import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

export async function GET(req) {
  const clientId = process.env.GITHUB_OAUTH_CLIENT_ID

  if (!clientId) {
    return new NextResponse('GITHUB_OAUTH_CLIENT_ID не задан в переменных окружения.', { status: 500 })
  }

  const siteUrl = process.env.SITE_URL || req.nextUrl.origin
  const redirectUri = `${siteUrl}/api/callback`
  const state = crypto.randomUUID()
  const authorizeUrl = new URL('https://github.com/login/oauth/authorize')

  authorizeUrl.searchParams.set('client_id', clientId)
  authorizeUrl.searchParams.set('redirect_uri', redirectUri)
  authorizeUrl.searchParams.set('scope', 'repo,user')
  authorizeUrl.searchParams.set('state', state)

  return NextResponse.redirect(authorizeUrl.toString())
}
