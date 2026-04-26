```sh
curl -X POST http://localhost:8080/v1/ingest \
  -H "X-Tenant-ID: test-tenant-1" \
  -F "context_text=Это тестовый текст для проверки работы Ingest-конвейера."
```

```sh
  echo "Содержимое тестового файла" > test.txt

  curl -X POST http://localhost:8080/v1/ingest \
    -H "X-Tenant-ID: test-tenant-1" \
    -F "context_text=Текст описания файла" \
    -F "file=@test.txt"
```

```sh
    curl -X POST http://localhost:8080/v1/search \
      -H "X-Tenant-ID: test-tenant-1" \
      -H "Content-Type: application/json" \
      -d '{"query_text": "тестовый текст", "top_k": 3}'
```
