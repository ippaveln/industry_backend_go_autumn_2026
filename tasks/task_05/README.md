# task_05: Generic-кэш с отказом при заполнении

![task 05](../../badges/tasks/task_05.svg)

Реализуйте `Cache[K comparable,V any]`, `NewCache[K,V](capacity int) *Cache[K,V]`, `Get(key K) (V,bool)`, `Set(key K,value V) bool`. Capacity — жёсткий предел числа ключей. Новый ключ в заполненном кэше отклоняется: false, состояние не меняется. Обновление существующего ключа разрешено даже при заполнении и возвращает true. Capacity <= 0 выключает кэш. Промах → zero V,false; сохранённое нулевое значение → zero V,true. Кэш создаётся конструктором, nil-получатель поддерживать не нужно. Потокобезопасность не требуется. Пример при capacity=1: Set(a,1) → true, Set(b,2) → false, Set(a,3) → true.

Меняйте только solution.go. Используйте stdlib. Проверка: `go test -race -count=4 ./tasks/task_05`.
