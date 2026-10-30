# go-musthave-shortener-tpl

## Профилирование памяти (Инкремент 17)

### Методика

Профили памяти снимались бенчмарками (`-memprofile`) для каждого пакета и объединялись через `go tool pprof -proto`:

```bash
go test -bench=. -benchmem -benchtime=1s -memprofile=profiles/raw/service.pprof -run=^$ ./internal/service
go test -bench=. -benchmem -benchtime=1s -memprofile=profiles/raw/audit.pprof -run=^$ ./internal/audit
go test -bench=. -benchmem -benchtime=1s -memprofile=profiles/raw/repo.pprof -run=^$ ./internal/repository
go test -bench=. -benchmem -benchtime=1s -memprofile=profiles/raw/handler.pprof -run=^$ ./internal/handler
go tool pprof -proto -output=profiles/base.pprof profiles/raw/*.pprof
```

Сравнение до/после:

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

### Оптимизации

1. **`sync.Pool` для `gzip.Writer`** — `gzip.NewWriter` выделяет ~300 КБ на каждый ответ. Пул возвращает writer в повторное использование, что снижает аллокации почти до нуля. Файл: `internal/middleware/gzip.go`.
2. **`sync.Pool` для `bytes.Buffer` в аудите** — снижает GC-давление на потоке событий аудита. Файлы: `internal/audit/buffers.go`, `internal/audit/http.go`.
3. **Preallocation слайсов и мап** — `make([]T, 0, 16)` в `GetByUserID` (все репозитории) и `make(map[K]V, len(items))` в `ShortenBatch` убирают realloc при большом количестве элементов.

### Результат diff

```
File: audit.test
Type: alloc_space
Time: 2026-10-28 21:58:29 MSK
Showing nodes accounting for -715.94MB, 12.17% of 5881.57MB total
Dropped 53 nodes (cum <= 29.41MB)
      flat  flat%   sum%        cum   cum%
 -424.89MB  7.22%  7.22%  -424.89MB  7.22%  github.com/Knapptan/Y-URL-shortener/internal/repository.(*FileRepository).GetByUserID
 -295.60MB  5.03% 12.25%  -295.60MB  5.03%  github.com/Knapptan/Y-URL-shortener/internal/repository.(*InMemoryRepo).GetByUserID
    7.05MB  0.12% 12.13%     8.56MB  0.15%  encoding/json.appendIndent
   -2.50MB 0.043% 12.17%     6.06MB   0.1%  github.com/Knapptan/Y-URL-shortener/internal/repository.(*FileRepository).saveToFile
         0     0% 12.17%     8.56MB  0.15%  encoding/json.(*Encoder).Encode
         0     0% 12.17%     6.06MB   0.1%  github.com/Knapptan/Y-URL-shortener/internal/repository.(*FileRepository).Save
         0     0% 12.17%  -418.84MB  7.12%  github.com/Knapptan/Y-URL-shortener/internal/repository.BenchmarkFileRepoGetByUserID
         0     0% 12.17%  -296.10MB  5.03%  github.com/Knapptan/Y-URL-shortener/internal/repository.BenchmarkInMemoryGetByUserID
         0     0% 12.17%  -718.08MB 12.21%  testing.(*B).launch
         0     0% 12.17%  -717.06MB 12.19%  testing.(*B).runN
```

Отрицательные значения в diff показывают снижение потребления памяти после оптимизаций.

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:

- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**
