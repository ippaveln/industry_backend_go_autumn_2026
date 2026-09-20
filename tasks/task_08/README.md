# task_08: Token bucket: атомарное списание

![task 08](../../badges/tasks/task_08.svg)

Объявите `Clock interface { Now() time.Time }`, тип Limiter, `NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter`, `AllowN(n int) bool`. Корзина изначально полна (burst токенов). Время только через Clock. За секунду добавляется ratePerSec токенов, включая дробную часть; запас ограничен burst. AllowN атомарно списывает ровно n токенов либо возвращает false **без частичного списания**. n<=0 и n>burst всегда false. Burst<=0 или nil Clock отключает лимитер. Rate<=0 не пополняет запас (rate конечный). При переводе часов назад запас не пополняется и последняя достигнутая отметка времени не уменьшается. Все вызовы AllowN потокобезопасны. Пример: burst=3, rate=0: AllowN(4) false; AllowN(2) true; AllowN(2) false; AllowN(1) true. Реальный time.Now и ожидания внутри лимитера запрещены.

Меняйте только solution.go. Используйте stdlib. Проверка: `go test -race -count=4 ./tasks/task_08`.
