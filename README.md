# pubsub-go

A production-ready Go pub/sub scaffold using **RabbitMQ** (topic exchange) with a **Mattermost** notification subscriber.

![pub/sub](image.png)

1. Publisher creates an event struct, marshals it to JSON, and publishes it to the ```events``` exchange with the routing key ```notification.general```.
2. Exchange (```events```, topic type) receives the message and matches the routing key against its bindings - ```notification.#``` matches anything starting with ```notification.```
3. Queue (```mattermost.notifications```) receives the routed message and holds it until a consumer picks it up.
4. Subscriber consumes from the queue, unmarshals the event, and calls the Mattermost webhook.

## Project structure

```
pubsub-go/
├── publisher/
│   ├── main.go          # Publishes sample events every 10 s
│   └── Dockerfile
├── subscriber/
│   ├── main.go          # Consumes events → Mattermost and Logwriters
│   └── Dockerfile
├── internal/
│   ├── config/          # Env-var config loader
│   ├── events/          # Shared Event type + routing-key constants
│   ├── mattermost/      # Incoming-webhook client
│   └── rabbitmq/        # AMQP client with reconnect logic
│   └── logwriter/       # Apps receiving events via logs
├── scripts/
│   ├── rabbitmq-definitions.json   # Pre-seeds exchange, queue & binding
│   └── enabled_plugins             # Enables the management plugin
├── docker-compose.yml
├── .env.example
└── go.mod
```

## Quick start

### 1. Prerequisites

- Docker ≥ 24 with the Compose plugin
- A Mattermost instance with an incoming webhook

### 2. Configure

- Add any necessary secrets/variables to the ```.env.example``` file. This file gets copied during startup as ```.env``` to be used by the apps.


### 3. Run

- Start the app.

```bash
make up
```

| Service    | URL                          | Notes                                      |
|------------|------------------------------|--------------------------------------------|
| RabbitMQ   | amqp://localhost:5672        | AMQP endpoint                              |
| Management | http://localhost:15672       | UI — login: ops_role / opsarethebest       |
| Publisher  | —                            | Publishes a heartbeat every 10 s           |
| Subscriber | —                            | Forwards every event to MM                 |

### 4. Tear down

```bash
make down
```

---

## Routing keys

| Constant               | Value                   | Used by    |
|------------------------|-------------------------|------------|
| `RouteNotification`    | `notification.general`  | Publisher  |
| `RouteAlert`           | `notification.alert`    | Publisher  |
| `RouteAll`             | `notification.#`        | Subscriber |

Add more routing keys in `internal/events/event.go` and additional queues/bindings in `scripts/rabbitmq-definitions.json`.

---

## Publishing your own events

```go
evt := events.Event{
    ID:       newID(),
    Type:     "deploy",
    Severity: events.SeverityInfo,
    Title:    "Deployment succeeded",
    Body:     "v2.3.1 deployed to production",
    Source:   "ci-pipeline",
    Timestamp: time.Now().UTC(),
    Meta: map[string]string{"app": "api-gateway"},
}
payload, _ := json.Marshal(evt)
rmq.Publish(ctx, routingKey, payload)
```

---

## Environment variables

Environment variables are located in the `.env.example` and `docker-compose.yml` files.  
App-specific variables are found in `docker-compose.yml`, while global variables like `LOG_LEVEL` are found in `.env.example`.

---

### Global Variables (`.env.example`)

| Variable                  | Required | Default              | Description                         |
|--------------------------|----------|----------------------|-------------------------------------|
| RABBITMQ_EXCHANGE        | No       | events               | Topic exchange name                 |
| RABBITMQ_DEFAULT_USER    | Yes      | ops_role             | RabbitMQ username                   |
| RABBITMQ_DEFAULT_PASS    | Yes      | opsarethebest        | RabbitMQ password                   |
| MATTERMOST_WEBHOOK_URL   | Yes      | —                    | Incoming webhook URL                |
| MATTERMOST_CHANNEL       | No       | @christian.graham    | Override destination channel        |
| MATTERMOST_USERNAME      | No       | @christian.graham    | Display name for messages           |
| MATTERMOST_ICON_URL      | No       | —                    | Bot avatar URL                      |
| LOG_LEVEL                | No       | info                 | debug / info / warn / error         |

---

### App-Specific Variables (`docker-compose.yml`)

| Variable                    | Service        | Default                                                                 | Description                                      |
|-----------------------------|----------------|-------------------------------------------------------------------------|--------------------------------------------------|
| RABBITMQ_URL               | All services   | amqp://${RABBITMQ_DEFAULT_USER}:${RABBITMQ_DEFAULT_PASS}@rabbitmq:5672/ | Full AMQP connection URL                         |
| RABBITMQ_QUEUE             | log-general    | log.general                                                             | Queue name for general logs                      |
| RABBITMQ_QUEUE             | log-alert      | log.alert                                                               | Queue name for alert logs                        |
| LOGWRITER_BINDING          | log-general    | notification.general                                                    | Routing key binding for general logs             |
| LOGWRITER_BINDING          | log-alert      | notification.alert                                                      | Routing key binding for alert logs               |
| APP_ENV                    | All services   | docker                                                                  | Application environment                          |
| RABBITMQ_DEFAULT_USER      | rabbitmq       | ${RABBITMQ_DEFAULT_USER}                                                | RabbitMQ broker username (from `.env.example`)   |
| RABBITMQ_DEFAULT_PASS      | rabbitmq       | ${RABBITMQ_DEFAULT_PASS}                                                | RabbitMQ broker password (from `.env.example`)   |
| RABBITMQ_LOAD_DEFINITIONS  | rabbitmq       | /etc/rabbitmq/definitions.json  

---

## Extending to other consumers

The project currently has two consumer types:

- **`subscriber`** — receives every `notification.#` event and forwards to Mattermost
- **`logwriter`** — receives events on a specific routing key and writes them to stdout (container logs)

To add a new consumer:

1. Add a new durable queue and binding in `scripts/rabbitmq-definitions.json` with the routing key(s) it should receive.
2. Create a new binary under its own directory (e.g. `slackwriter/main.go`), or if it's purely stdout-based, add a new service in `docker-compose.yml` reusing the `logwriter` image with a different `RABBITMQ_QUEUE` and `LOGWRITER_BINDING`.
3. If the consumer needs its own client (Slack, PagerDuty, email, etc.), add a client package under `internal/` and instantiate it in the new binary's `main.go` — following the same pattern as `internal/mattermost`.
4. Add a build target in the `subscriber/Dockerfile` and a corresponding `make` target in the `Makefile`.

Note that `config.RequireMattermost()` is called exclusively by the `subscriber` binary. New binaries that don't use Mattermost should not call it — add an equivalent `RequireX()` method to `internal/config/config.go` for any credentials their consumer needs.