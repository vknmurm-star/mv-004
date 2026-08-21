import { siteUrl } from '../lib/content'
const geoCrawlers = ['GPTBot','OAI-SearchBot','ChatGPT-User','PerplexityBot','ClaudeBot','anthropic-ai','Google-Extended','Amazonbot','Bytespider','CCBot']
export default function robots(){return {rules:[{userAgent:'*',allow:'/',disallow:['/admin','/api']},...geoCrawlers.map(userAgent=>({userAgent,allow:'/'}))],sitemap:`${siteUrl}/sitemap.xml`,host:siteUrl}}
