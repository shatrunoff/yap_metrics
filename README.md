# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
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


## Профайлинг
# Diff профилей
➜  yap_metrics git:(iter17) ✗ go tool pprof -top -diff_base=profiles/profile_before.pprof  profiles/profile_after.pprof
File: server
Type: inuse_space
Time: 2025-11-29 21:00:16 MSK
Showing nodes accounting for 244.92kB, 7.94% of 3085.96kB total
Dropped 6 nodes (cum <= 15.43kB)
      flat  flat%   sum%        cum   cum%
-1031.14kB 33.41% 33.41% -1031.14kB 33.41%  reflect.growslice
  768.26kB 24.90%  8.52%   768.26kB 24.90%  go.uber.org/zap/zapcore.newCounters (inline)
 -516.76kB 16.75% 25.26%  -516.76kB 16.75%  runtime.procresize
  512.56kB 16.61%  8.65%   512.56kB 16.61%  runtime.makeProfStackFP (inline)
  512.05kB 16.59%  7.94%  1280.31kB 41.49%  runtime.main
 -512.05kB 16.59%  8.65%  -512.05kB 16.59%  time.NewTicker
  512.01kB 16.59%  7.94%   512.01kB 16.59%  reflect.New
         0     0%  7.94%  -519.13kB 16.82%  encoding/json.(*Decoder).Decode
         0     0%  7.94%  -519.13kB 16.82%  encoding/json.(*decodeState).array
         0     0%  7.94%   512.01kB 16.59%  encoding/json.(*decodeState).literalStore
         0     0%  7.94%   512.01kB 16.59%  encoding/json.(*decodeState).object
         0     0%  7.94%  -519.13kB 16.82%  encoding/json.(*decodeState).unmarshal
         0     0%  7.94%  -519.13kB 16.82%  encoding/json.(*decodeState).value
         0     0%  7.94%   512.01kB 16.59%  encoding/json.indirect
         0     0%  7.94%  -519.13kB 16.82%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0%  7.94%  -519.13kB 16.82%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0%  7.94%  -519.13kB 16.82%  github.com/shatrunoff/yap_metrics/internal/handler.(*Handler).updateMetricsBatch
         0     0%  7.94%   768.26kB 24.90%  github.com/shatrunoff/yap_metrics/internal/handler.NewHandler
         0     0%  7.94%  -519.13kB 16.82%  github.com/shatrunoff/yap_metrics/internal/middleware.GzipCompressionMiddleware.func1
         0     0%  7.94%  -519.13kB 16.82%  github.com/shatrunoff/yap_metrics/internal/middleware.GzipDecompressionMiddleware.func1
         0     0%  7.94%   768.26kB 24.90%  github.com/shatrunoff/yap_metrics/internal/middleware.InitLogger
         0     0%  7.94%  -519.13kB 16.82%  github.com/shatrunoff/yap_metrics/internal/middleware.LoggingMiddleware.func1
         0     0%  7.94%  -512.05kB 16.59%  github.com/shatrunoff/yap_metrics/internal/service.(*FileStorageService).Start.func1
         0     0%  7.94%  -512.05kB 16.59%  github.com/shatrunoff/yap_metrics/internal/service.(*FileStorageService)startPeriodicSave
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.(*Logger).WithOptions
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.Config.Build
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.Config.buildOptions.WrapCore.func5
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.Config.buildOptions.func1
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.New
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.NewProduction
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap.optionFunc.apply
         0     0%  7.94%   768.26kB 24.90%  go.uber.org/zap/zapcore.NewSamplerWithOptions
         0     0%  7.94%   768.26kB 24.90%  main.initServer
         0     0%  7.94%   768.26kB 24.90%  main.main
         0     0%  7.94%  -519.13kB 16.82%  net/http.(*conn).serve
         0     0%  7.94%  -519.13kB 16.82%  net/http.HandlerFunc.ServeHTTP
         0     0%  7.94%  -519.13kB 16.82%  net/http.serverHandler.ServeHTTP
         0     0%  7.94% -1031.14kB 33.41%  reflect.Value.Grow
         0     0%  7.94% -1031.14kB 33.41%  reflect.Value.grow
         0     0%  7.94%   512.56kB 16.61%  runtime.findRunnable
         0     0%  7.94%   512.56kB 16.61%  runtime.injectglist
         0     0%  7.94%   512.56kB 16.61%  runtime.injectglist.func1
         0     0%  7.94%   512.56kB 16.61%  runtime.mProfStackInit (inline)
         0     0%  7.94%   512.56kB 16.61%  runtime.mcommoninit
         0     0%  7.94%     -513kB 16.62%  runtime.resetspinning
         0     0%  7.94%  -516.76kB 16.75%  runtime.rt0_go
         0     0%  7.94%  -516.76kB 16.75%  runtime.schedinit
         0     0%  7.94%     -513kB 16.62%  runtime.wakep