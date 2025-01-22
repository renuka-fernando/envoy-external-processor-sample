### Run Setup

Start Ext Processing Server:
```sh
cd external_processor; go run main.go; cd -
```

Start Envoy and the backend service:
```sh
docker compose up -d; docker compose logs -ft
```

### Test

```sh
curl http://localhost:8000/foo/123?myQuery=bar -H 'MyHeader: Foo' -i -d 'foo bar'
```

### Stop

```sh
docker compose down
```
