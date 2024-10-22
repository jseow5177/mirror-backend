# CDP

## Init Local Dev Env

```bash
brew install docker

# Start middlewares
docker compose -f ./scripts/compose.yaml up

# Start Go server
go run main.go

# Close all middlewares
docker compose -f ./scripts/compose.yaml down
```

## Kafka UI

We use [Provectus Kafka UI](https://github.com/provectus/kafka-ui).

Access at http://localhost:8080.

