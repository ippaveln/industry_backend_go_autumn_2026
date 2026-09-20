# Мини “production-like” сервис: in-memory repository + HTTP API + конкурентная безопасность

![task 10](../../badges/tasks/task_10.svg)

Сделайте небольшой сервис задач: in-memory репозиторий + HTTP API. Решение будет проверяться тестами, поэтому важно строго следовать контракту ниже.

## Доменная модель


type Task struct {
 ID        string
 Title     string
 Done      bool
 UpdatedAt time.Time
}


## Контракт хранилища


type TaskRepo interface {
 Create(title string) (Task, error)
 Get(id string) (Task, bool)
 List(done bool) []Task
 SetDone(id string, done bool) (Task, error)
}


## HTTP API

### Эндпоинты

- POST /tasks → создать задачу
- GET /tasks/{id} → получить задачу по id
- GET /tasks?done=false → получить незавершённые задачи; done=true → завершённые
- PATCH /tasks/{id} → обновить done (true/false)

### Форматы запросов

POST /tasks:


{"title":"buy milk"}


PATCH /tasks/{id}:


{"done":true}


### Форматы ответов

- Одна задача возвращается JSON-объектом с полями: id, title, done, updatedAt
- GET /tasks возвращает JSON-массив задач

## Требования к поведению

### 1) Потокобезопасность репозитория

- Реализация in-memory (например, map[string]Task)
- Защита через sync.RWMutex
- List(done) должен возвращать копию данных (чтобы внешний код не мог менять внутреннее состояние)

### 2) Время — только через внедрение часов

- Нельзя вызывать time.Now() напрямую внутри репозитория
- Сделайте интерфейс часов и внедрите его в репозиторий, например:


type Clock interface { Now() time.Time }


- UpdatedAt выставляется при Create и обновляется при SetDone только если Done действительно изменился. Повторная установка того же Done возвращает прежнюю задачу без изменения UpdatedAt.

### 3) Валидация JSON и входных данных

- Невалидный JSON → 400 Bad Request
- Отсутствующие/неподходящие поля → 400 Bad Request
- Рекомендуется использовать json.Decoder + DisallowUnknownFields()
- title должен быть непустым после strings.TrimSpace

### 4) Корректные HTTP статусы

- POST /tasks → 201 Created
- Успешные GET / PATCH → 200 OK
- Не найдено (несуществующий id) → 404 Not Found
- Ошибка ввода/валидации → 400 Bad Request

### 5) Детерминированность списка

Чтобы ответы были стабильными, GET /tasks должен возвращать задачи в предсказуемом порядке:

- сортировка по UpdatedAt по убыванию (новые сначала)
- при равенстве UpdatedAt — по ID по возрастанию

### 6) ID

- ID должен быть уникальным в рамках процесса (можно счётчик/UUID — на ваше усмотрение)
- Пустой ID недопустим
## Точный осенний контракт

- Конструкторы: `NewInMemoryTaskRepo(clock Clock) TaskRepo` и `NewHTTPHandler(repo TaskRepo) http.Handler`. Clock не nil; `Clock interface { Now() time.Time }`.
- `List(done bool)` возвращает только задачи с соответствующим Done; пустой результат — непустой по представлению JSON массив `[]`, не `null`. Полученный срез не разделяет изменяемые данные с репозиторием.
- GET /tasks без параметра done равнозначен done=false. Допустимы только строки true/false; пустой, повторный или другой done → 400. Другие query-параметры игнорируются.
- Title хранится после strings.TrimSpace. ErrInvalidTitle и ErrNotFound — sentinel-ошибки, проверяемые errors.Is. Ошибки репозитория не меняют состояние.
- POST и PATCH принимают ровно один JSON-объект. Неизвестные поля, null вместо обязательного значения, отсутствующее значение, неверный тип, дополнительный JSON/мусор → 400. Пробелы после JSON допустимы. Повторные известные JSON-поля обрабатываются стандартным encoding/json (последнее значение).
- Успешные ответы имеют Content-Type application/json. Неизвестный путь, пустой id или вложенный путь → 404; неподдерживаемый метод на известном пути → 405.
- NewHTTPHandler работает через переданный интерфейс TaskRepo. Неизвестная ошибка Create/SetDone → 500, ErrInvalidTitle → 400, ErrNotFound из SetDone → 404. Формат тела ошибки не фиксируется.
- Пример: создать задачу, PATCH done=true, повторить PATCH позднее — UpdatedAt не меняется; GET /tasks возвращает [], GET /tasks?done=true возвращает задачу.
