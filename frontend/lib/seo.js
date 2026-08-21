import { absolute, siteUrl } from './content'

export function siteJsonLd() {
  return {
    '@context': 'https://schema.org',
    '@graph': [
      {
        '@type': 'Organization',
        '@id': `${siteUrl}/#organization`,
        name: 'На высоте',
        url: siteUrl,
        logo: { '@type': 'ImageObject', url: absolute('/favicon.svg') },
      },
      {
        '@type': 'WebSite',
        '@id': `${siteUrl}/#website`,
        name: 'На высоте',
        url: siteUrl,
        inLanguage: 'ru-RU',
        publisher: { '@id': `${siteUrl}/#organization` },
        potentialAction: {
          '@type': 'SearchAction',
          target: `${siteUrl}/search?q={search_term_string}`,
          'query-input': 'required name=search_term_string',
        },
      },
    ],
  }
}
