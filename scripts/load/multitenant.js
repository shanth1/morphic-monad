import http from 'k6/http'

export const options = { vus: 50, duration: '1m' }

export default function () {
  const randomTenant = `tenant_${Math.floor(Math.random() * 1000000)}`
  const payload = JSON.stringify({ query_text: 'поиск', top_k: 2 })
  http.post('http://localhost:8080/v1/search', payload, {
    headers: { 'X-Tenant-ID': randomTenant, 'Content-Type': 'application/json' },
  })
}
