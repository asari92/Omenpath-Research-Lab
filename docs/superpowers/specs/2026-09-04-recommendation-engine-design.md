# Recommendation Engine — дизайн для Block C

## 1. Цель

Recommendation в Portal Details должна давать одно полезное и исполнимое
направление действия, не заменяя domain rules и не выполняя command
автоматически. Backend реализует расчёт в Stage 12; frontend Stage 17 только
показывает полученный enum.

Приоритет продукта:

```text
сохранить Observer
→ избежать Collapse и обнуления Laboratory Energy
→ продвинуть исследование нового Plane
→ не тратить ресурсы на бесполезное повторное исследование
```

## 2. Граница и API

Чистая domain-функция получает текущий aggregate, Portal ID, `now` и config и
возвращает значение закрытого enum либо отсутствие значения для terminal
Portal:

```go
type Recommendation string

const (
    RecommendationLeaveOpen       Recommendation = "LEAVE OPEN"
    RecommendationWaitForCorridor Recommendation = "WAIT FOR CORRIDOR"
    RecommendationStabilize       Recommendation = "STABILIZE"
    RecommendationClose           Recommendation = "CLOSE"
    RecommendationSendObserver    Recommendation = "SEND OBSERVER"
    RecommendationRecallObserver  Recommendation = "RECALL OBSERVER"
)

func RecommendationForPortal(
    state SimulationState,
    portalID int64,
    now time.Time,
    cfg config.Config,
) (Recommendation, bool, error)
```

`bool=false` означает terminal Portal и сериализуется в Portal Details как
`recommendation: null`. Неизвестный Portal и нарушенный aggregate возвращают
ошибку.

Функция:

- не мутирует state;
- не вызывает random;
- не читает hidden instability timestamp;
- не используется командами как дополнительный запрет;
- вызывается после manager `ResolveTick(now)`, поэтому работает с актуальным
  состоянием.

## 3. Безопасный временной горизонт

Engine не требует от игрока сравнивать таймеры вручную. Он консервативно
проверяет, что Portal останется пригодным на весь требуемый период.

```text
SEND/RECALL сейчас       ObserverTransitMax
WAIT FOR CORRIDOR        creature_clearance_remaining + ObserverTransitMax
активный transit         observer.phase_ends_at - now
EXPLORING → return       research_remaining + ObserverTransitMax
```

Текущий путь безопасен, когда Portal STABLE, не CRITICAL и его
`effective_lifetime` строго больше требуемого horizon. При равенстве Portal
lifecycle разрешается раньше Observer lifecycle и Observer станет LOST. Для
active transit используется его уже известный deadline; новый random transit
draw не выполняется.

Для WAIT используется существующая derived-механика creatures. Если после
очистки на максимальный transit времени не останется, `WAIT FOR CORRIDOR` не
предлагается.

## 4. Hypothetical Stabilize

`STABILIZE` предлагается не просто из-за флага UNSTABLE. На копии Portal
проверяется обычный `Stabilize(now, cfg)`, а затем повторно вычисляется safety
horizon. Дополнительно учитывается доступность 20 Laboratory Energy либо
бесплатное действие во время Leyline Override.

Recommendation `STABILIZE` допустима только если:

1. Portal OPEN и UNSTABLE;
2. current Portal Energy не превышает 85%;
3. Laboratory Energy хватает на текущую effective cost;
4. hypothetical stabilized Portal реально обеспечивает нужный horizon.

Копия отбрасывается после расчёта. Реальные Portal/Lab/Observer не меняются.

## 5. Decision table

Правила применяются сверху вниз.

### 5.1 Extraction synchronization

OPEN Extraction Portal до synchronization получает `LEAVE OPEN`. Sync — часть
lifecycle; Recommendation не предлагает невозможный ранний RECALL.

### 5.2 Active transit

Если Observer уже движется через этот Portal:

1. current path безопасен → `LEAVE OPEN`;
2. hypothetical Stabilize делает его безопасным → `STABILIZE`;
3. иначе → `LEAVE OPEN`.

`CLOSE` не предлагается: он гарантированно превратит такого Observer в LOST.

### 5.3 WAITING_RETURN

Если в destination Plane есть WAITING_RETURN Observer и flow равен NONE или
INBOUND:

1. corridor свободен и RECALL безопасен → `RECALL OBSERVER`;
2. creatures ещё идут, но clearance + max transit помещаются в безопасный
   horizon → `WAIT FOR CORRIDOR`;
3. hypothetical Stabilize делает return безопасным → `STABILIZE`;
4. иначе применяется safety fallback из §5.6.

При нескольких ожидающих используется уже определённый gameplay rule:
longest-waiting, затем lower Observer ID. При flow NONE возврат имеет приоритет
над отправкой нового Observer.

### 5.4 EXPLORING

Если Observer исследует destination Plane, а flow NONE или INBOUND может быть
использован для будущего возврата:

1. Portal безопасно переживёт остаток research и max return transit →
   `LEAVE OPEN`;
2. hypothetical Stabilize создаст нужный запас → `STABILIZE`;
3. иначе применяется safety fallback.

OUTBOUND Portal не стабилизируется только ради будущего RECALL: его направление
для этого непригодно.

### 5.5 Новый SEND

SEND рассматривается только когда Plane UNEXPLORED, в нём нет EXPLORING,
WAITING_RETURN или RETURNING Observer, к нему не движется OUTBOUND Observer
через другой Portal, flow NONE/OUTBOUND и существует AVAILABLE Observer:

1. corridor свободен и max transit безопасен → `SEND OBSERVER`;
2. creatures блокируют путь, но clearance + max transit безопасны →
   `WAIT FOR CORRIDOR`;
3. hypothetical Stabilize делает SEND безопасным → `STABILIZE`;
4. иначе применяется safety fallback.

В EXPLORED Plane новый Observer не отправляется. Второй Observer не
рекомендуется, пока в тот же Plane уже идёт outbound transit, exploration,
ожидание или возврат.

### 5.6 Close/leave fallback

- EXPLORED Plane без Observer → `CLOSE`, если ordinary Close command доступен;
  иначе `LEAVE OPEN`.
- Для остальных Portal без безопасной mission action:
  HIGH/CRITICAL → `CLOSE`, если Close доступен; иначе `LEAVE OPEN`.
- LOW/MEDIUM → `LEAVE OPEN`.

Close availability учитывает Laboratory Energy и Leyline Override. Creatures
могут потребовать UI confirmation, но это не исключает `CLOSE` из рекомендации.
Active transit никогда не доходит до этого fallback.

## 6. Transport и UI boundary

Stage 12 добавляет в Portal Details:

```json
{"recommendation": "RECALL OBSERVER"}
```

Для terminal Portal:

```json
{"recommendation": null}
```

Dashboard slot, state summary и WebSocket slot DTO Recommendation не содержат.
Если WebSocket передаёт полный Portal Details в будущем, действует та же
visibility boundary: поле принадлежит только Details view model.

## 7. Ошибки и детерминизм

- Invalid aggregate/enum/reference → domain invariant error.
- Unknown Portal → stable not-found error manager/transport boundary.
- Недостаток Laboratory Energy не является ошибкой вычисления: соответствующая
  action просто не выбирается.
- Recommendation не создаёт Events и не влияет на persistence.
- При одинаковом state/config/now результат всегда одинаков.

## 8. Проверка Stage 12

Минимальный набор table-driven tests:

- terminal → null;
- Extraction before sync → LEAVE OPEN;
- active safe transit → LEAVE OPEN;
- active unsafe transit → STABILIZE только если это реально помогает;
- waiting Observer → RECALL, creatures → WAIT, unsafe → STABILIZE/fallback;
- longest-waiting selection не меняется;
- exploring Observer сохраняет пригодный inbound path;
- outbound path не сохраняется ради recall;
- unexplored empty Plane → safe SEND;
- explored Plane/другой outbound или Observer в Plane → SEND не предлагается;
- exact Portal/Observer deadline tie считается unsafe;
- HIGH/CRITICAL fallback → CLOSE, insufficient Lab Energy → LEAVE OPEN;
- hypothetical Stabilize не мутирует aggregate и не расходует random;
- hidden instability timestamp не влияет на result;
- REST Details содержит enum/null, Dashboard/slots не содержат поле;
- Recommendation никогда не меняет результат domain command.
