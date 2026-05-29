# Развёртывание в Kubernetes (minikube)

Локальный запуск `warehouse-controller` в Kubernetes. Само приложение — через
манифесты в этой папке, Postgres и Redis — через Helm-чарты Bitnami.

- Приложение: `Deployment` + `Service` (+ `Ingress`), конфиг через `ConfigMap` и `Secret`.
- Postgres: `helm install bitnami/postgresql` (StatefulSet + Service).
- Redis: `helm install bitnami/redis` (standalone, без пароля).
- Образ берётся из GHCR (`ghcr.io/uber23541/warehouse-controller`), куда его публикует CI.

## Предварительные требования

`minikube`, `kubectl`, `helm`.

## Быстрый старт

Все шаги ниже завёрнуты в цели Makefile (см. ниже), либо выполняются вручную.

### 1. Кластер и ingress

```bash
minikube start
minikube addons enable ingress
```

### 2. Helm-репозиторий Bitnami

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

### 3. Namespace и Secret

```bash
kubectl apply -f deploy/k8s/namespace.yaml

# Скопируйте шаблон, впишите реальные значения (JWT_SECRET, DB_PASSWORD,
# и тот же пароль внутри DB_DSN), затем примените:
cp deploy/k8s/secret.example.yaml deploy/k8s/secret.yaml

kubectl apply -f deploy/k8s/secret.yaml
```

`secret.yaml` в git не попадает (см. `.gitignore`).

### 4. Postgres и Redis (Helm)

Имена релизов важны — на них ссылаются `DB_DSN` и `REDIS_URL`:

```bash
helm install warehouse-postgresql bitnami/postgresql \
  -n warehouse -f deploy/k8s/helm/postgres-values.yaml

helm install warehouse-redis bitnami/redis \
  -n warehouse -f deploy/k8s/helm/redis-values.yaml
```

Postgres переиспользует наш Secret `warehouse-secrets` (`auth.existingSecret`),
поэтому пароль задаётся в одном месте.

### 5. Приложение

```bash
kubectl apply -k deploy/k8s
```

### 6. Проверка

```bash
kubectl get pods -n warehouse
kubectl logs deploy/warehouse-controller -n warehouse
```

Приложение применяет миграции при старте. Пока Postgres ещё не готов, под
может пару раз перезапуститься (CrashLoopBackOff) — это нормально, он
самовосстановится, как только БД поднимется.

## Доступ к API

**Вариант А — Ingress.** Узнайте IP и добавьте host:

```bash
minikube ip                       # например 192.168.49.2
# в hosts: «192.168.49.2 warehouse.local»
curl http://warehouse.local/health
```

**Вариант Б — без Ingress.** minikube сам откроет URL:

```bash
minikube service warehouse-controller -n warehouse
```

## Образ в GHCR

Пакет `ghcr.io/uber23541/warehouse-controller` публичный, поэтому pull-secret не нужен —
кластер тянет образ без аутентификации.

Сделать пакет публичным: GitHub → пакет `warehouse-controller` → **Package settings** →
**Change visibility** → **Public** (один раз после первой публикации образа CI).

## Удаление

```bash
kubectl delete -k deploy/k8s
helm uninstall warehouse-postgresql warehouse-redis -n warehouse
kubectl delete namespace warehouse
```
