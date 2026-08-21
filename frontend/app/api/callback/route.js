import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

function renderAuthScript(status, payload) {
  const message = status === 'success'
    ? `authorization:github:success:${JSON.stringify({ token: payload, provider: 'github' })}`
    : `authorization:github:error:${JSON.stringify({ message: payload })}`

  return `<!DOCTYPE html>
<html>
  <body>
    <script>
      (function () {
        function receiveMessage(e) {
          window.opener.postMessage(
            ${JSON.stringify(message)},
            e.origin
          );
          window.removeEventListener('message', receiveMessage, false);
        }
        window.addEventListener('message', receiveMessage, false);
        window.opener.postMessage('authorizing:github', '*');
      })();
    </script>
  </body>
</html>`
}

export async function GET(req) {
  const code = req.nextUrl.searchParams.get('code')
  const clientId = process.env.GITHUB_OAUTH_CLIENT_ID
  const clientSecret = process.env.GITHUB_OAUTH_CLIENT_SECRET

  if (!code || !clientId || !clientSecret) {
    return new NextResponse(renderAuthScript('error', 'Отсутствует code или переменные окружения OAuth.'), {
      headers: { 'Content-Type': 'text/html' },
    })
  }

  const tokenRes = await fetch('https://github.com/login/oauth/access_token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({ client_id: clientId, client_secret: clientSecret, code }),
  })
  const data = await tokenRes.json()

  if (!data.access_token) {
    return new NextResponse(renderAuthScript('error', data.error_description ?? 'Не удалось получить токен GitHub.'), {
      headers: { 'Content-Type': 'text/html' },
    })
  }

  return new NextResponse(renderAuthScript('success', data.access_token), {
    headers: { 'Content-Type': 'text/html' },
  })
}
