# prometheus-alertmanager-silencer
Allow silencing alerts based on maintenance schedule config. Status board can be reached via http on `/`.

## config
```yaml
maintenances:
  - matchers:
      - "alertname=test"
    schedule: "6 * * * *"
    duration: "1s"

  - matchers:
      - "alertname=test2"
    schedule: "* * * * *"
    duration: "60s"
```

## status board
```yaml
maintenance:
  matchers:
    - alertname=test
  schedule: 6 * * * *
  duration: 1s
next: 2021-04-07T04:06:00+07:00
isActive: false
---
maintenance:
  matchers:
    - alertname=test2
  schedule: '* * * * *'
  duration: 60s
next: 2021-04-07T03:09:00+07:00
isActive: true
```

## dependencies
* golang 1.13+

## dev-dependencies
* docker
* docker-compose
* make

## dev
* `make dev-up` starts dev-infrastructure
* `make lint`
* `make test` runs unit tests
* `make tests-with-infrastructure` runs integration tests with infrastructure. Require starting dev infrastructure first.

## deploy
checkout to required tag and run deploy
```
git checkout <tag>
make deploy
```
