import http from 'k6/http'

export const options = {
  scenarios: {
    ingest_load: {
      executor: 'constant-arrival-rate',
      rate: 300,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 100,
      maxVUs: 500,
      exec: 'runIngest',
    },
    search_load: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 100,
      maxVUs: 1000,
      exec: 'runSearch',
    },
  },
}

export function runIngest() {
  const tenantId = `tenant_${Math.floor(Math.random() * 10)}`
  const data = {
    context_text: 'Текст для векторизации',
    file: http.file(new ArrayBuffer(10), 'dummy.txt', 'text/plain'),
  }
  http.post('http://localhost:8080/v1/ingest', data, { headers: { 'X-Tenant-ID': tenantId } })
}

export function runSearch() {
  const tenantId = `tenant_${Math.floor(Math.random() * 10)}`
  const payload = JSON.stringify({ query_text: 'поиск', top_k: 3 })
  http.post('http://localhost:8080/v1/search', payload, {
    headers: { 'X-Tenant-ID': tenantId, 'Content-Type': 'application/json' },
  })
}
