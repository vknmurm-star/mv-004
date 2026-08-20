import { siteUrl } from '../lib/content'
export default function robots(){return {rules:[{userAgent:'*',allow:'/'},{userAgent:'GPTBot',allow:'/'},{userAgent:'PerplexityBot',allow:'/'},{userAgent:'ClaudeBot',allow:'/'}],sitemap:`${siteUrl}/sitemap.xml`,host:siteUrl}}
