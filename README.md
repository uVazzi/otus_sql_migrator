# gomigrator

![Go](https://img.shields.io/badge/go-1.24-blue)
![Build Status](https://github.com/uVazzi/otus_sql_migrator/actions/workflows/checks.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/uVazzi/otus_sql_migrator)](https://goreportcard.com/report/github.com/uVazzi/otus_sql_migrator)

Gomigrator — это простой инструмент миграции базы данных для PostgreSQL.

### Конфигурация

```
Переменные окружения:
DB_DSN - PostgreSQL DSN
MIGRATIONS_DIR - Path to migrations directory

Флаги:
dsn - PostgreSQL DSN
dir - Path to migrations directory
config - ath to configuration file
```

### Команды:

* `gomigrator create <имя_миграции>` - Создание миграции
* `gomigrator up` - Применение всех миграций
* `gomigrator down` - Откат последней миграции
* `gomigrator redo` - Повтор последней миграции (откат + накат)
* `gomigrator status` - Вывод статуса миграций
* `gomigrator dbversion` - Вывод версии базы
