# Саммари папки Google Drive (Go)

Программа создаёт общий саммари по содержимому папки: извлекает документы (PDF, DOCX, TXT, MD), разбивает на чанки, саммаризирует через LLM (OpenRouter) и формирует отчёт в Markdown или JSON.

Поддерживаются источники: **Google Drive** и **локальная папка** (см. `config.yaml`).

## Требования

- Go 1.21+
- (Для Drive) Google Cloud проект с включённым Drive API и OAuth credentials

## Установка

```bash
git clone <repo>
go build -o summary ./cmd/summary
# или запуск без сборки:
go run ./cmd/summary --help
```

## Настройка OpenRouter

1. Зарегистрируйтесь на [OpenRouter](https://openrouter.ai/).
2. Получите API Key: [Keys](https://openrouter.ai/keys).
3. Скопируйте `.env.example` в `.env` и укажите ключ:
   ```bash
   cp .env.example .env
   # В .env задайте OPENROUTER_API_KEY=sk-or-v1-...
   ```
4. Модель задаётся в `config.yaml` (поле `model`) или в `.env` (`OPENROUTER_MODEL`). По умолчанию в коде: `deepseek/deepseek-v3.2`. Актуальный список: [OpenRouter Models](https://openrouter.ai/models).

## Настройка Google Drive (при source: drive)

1. Создайте проект в [Google Cloud Console](https://console.cloud.google.com/).
2. Включите **Google Drive API**.
3. Создайте **OAuth 2.0 Client ID** (тип: Desktop app).
4. Скачайте JSON и сохраните как `credentials.json` в корне проекта.
5. В `.env` укажите `GOOGLE_DRIVE_FOLDER_ID` — ID папки из ссылки Drive (например `https://drive.google.com/drive/folders/ID` → `ID`).

При первом запуске `ingest` откроется браузер для авторизации; токен сохранится в `token.json`.

## Конфигурация

- **`.env`** — секреты и переопределения (ключи, пути). См. `.env.example`.
- **`config.yaml`** — режим, источник, лимиты (mode, source, chunk_tokens, llm_concurrency, model). Значения из `.env` имеют приоритет.

## Переменные (.env.example)

Скопируйте и заполните:

```bash
cp .env.example .env
```

| Переменная | Описание |
|------------|----------|
| `OPENROUTER_API_KEY` | API ключ OpenRouter (обязателен для summarize) |
| `OPENROUTER_MODEL` | Модель LLM (опционально, иначе из config.yaml) |
| `GOOGLE_CREDENTIALS_PATH` | Путь к `credentials.json` |
| `GOOGLE_DRIVE_FOLDER_ID` | ID папки Drive (для source: drive) |
| `GOOGLE_TOKEN_PATH` | Путь к файлу токена (по умолчанию `token.json`) |
| `CACHE_DIR` | Каталог кеша (по умолчанию `.cache`) |

## Примеры команд

```bash
# Скачать файлы из папки Drive
go run ./cmd/summary ingest --folder-id=1x6EKNkVw6PlFVTr6cGrsVscmRuwqGrXd

# С перекачиванием (игнорировать кеш)
go run ./cmd/summary ingest --force

# Ограничить число файлов (тест)
go run ./cmd/summary ingest --max-files 5

# Саммаризация (после ingest)
go run ./cmd/summary summarize --mode=fast

# Собрать отчёт в Markdown
go run ./cmd/summary report --format=md --output=summary.md

# Собрать отчёт в JSON
go run ./cmd/summary report --format=json --output=summary.json

# Полный цикл: ingest → summarize → report
go run ./cmd/summary
```

## Подкоманды и флаги

| Подкоманда | Описание |
|------------|----------|
| `ingest` | Скачать файлы из источника (Drive или local по config) |
| `summarize` | Саммаризация через LLM, результат в `.cache/summary_result.json` |
| `report` | Собрать отчёт из кеша в файл (md/json) |

Без подкоманды выполняется цепочка: **ingest → summarize → report** (вывод по умолчанию в `summary.md`).

| Флаг | Описание |
|------|----------|
| `--cache-dir` | Каталог кеша |
| `--folder-id` | ID папки Drive (переопределяет .env) |
| `--verbose` | Подробный вывод |
| `--max-files` | Макс. число файлов (0 = без лимита) |
| `--force` | Перекачать файлы при ingest |
| `--mode` | Режим саммаризации: fast \| deep |
| `--format` | Формат отчёта: md \| json |
| `--output` | Файл вывода отчёта |

## Как создаётся и обновляется summary.md

1. **ingest** — скачивает файлы в `.cache/downloads/`, сохраняет список в `.cache/file_list.json`. Файл `summary.md` на этом шаге **не создаётся**.

2. **summarize** — загружает тексты, режет на чанки, вызывает LLM, собирает отчёт в память и сохраняет его в **`.cache/summary_result.json`**. Файл `summary.md` по-прежнему **не создаётся**.

3. **report** — читает `.cache/summary_result.json` и записывает отчёт в файл из `--output` (по умолчанию **`summary.md`**). Здесь **создаётся или перезаписывается** `summary.md`.

При запуске **без подкоманды** выполняются все три шага подряд; в конце `report` пишет в `summary.md` (или в путь из `--output`).

### Проверка локально

```bash
cd go-py-test1

# Вариант А: полный цикл (ingest → summarize → report)
go run ./cmd/summary
# В конце должен появиться/обновиться summary.md

# Вариант Б: по шагам (если ingest и summarize уже делали)
go run ./cmd/summary report --output=summary.md --format=md
# summary.md создаётся/обновляется из .cache/summary_result.json

# Убедиться, что файл есть и не пустой
ls -la summary.md
head -20 summary.md
```

При каждом успешном запуске `report` (или полного цикла) содержимое `summary.md` **полностью перезаписывается** из текущего результата саммаризации.

## Локальный источник

В `config.yaml` задайте `source: local` и при необходимости `local_dir: ./путь/к/папке`. Затем:

```bash
go run ./cmd/summary ingest
```

Файлы будут браться из указанной папки без обращения к Drive.
