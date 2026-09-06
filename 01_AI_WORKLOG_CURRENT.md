# AI Worklog — Current

## Текущее состояние

На 2026-09-06:

- Blocks A–D имеют статус GREEN, включая corrective gate блока D.
- Block E имеет статус GREEN в отдельно согласованном минимальном deployment scope.
- Приложение развёрнуто за существующим nginx/HTTPS.
- Последние UI-коррекции реализованы до коммита 27a9529 включительно.
- Отложенные проверки и улучшения перечислены ниже и не выдаются за выполненные.

Актуальные источники подробностей:

- product semantics — 00_FINAL_SPEC_v5.md;
- порядок стадий и границы блоков — 02_IMPLEMENTATION_ROADMAP_TDD.md;
- покрытие требований — docs/traceability.md;
- запуск на сервере — docs/deployment.md;
- полная RED/GREEN история — Git log и соответствующие stage/block plans.

## Время разработки, токены и AI tools

Активная разработка шла с 2026-09-01 по 2026-09-06. Точное число рабочих часов
не фиксировалось, поэтому оценка не выдумывается.

Интерфейс не предоставил надёжную суммарную статистику токенов за весь проект.
Token usage недоступен.

Использованные инструменты:

- ChatGPT/Codex — анализ, спецификация, код, review, debugging, документация,
  генерация assets и deployment guidance;
- Git — checkpoint-level RED/GREEN evidence и история corrective passes;
- Go toolchain — format, vet, build, unit/integration и race tests;
- React, TypeScript, Vite, Vitest, Testing Library и Playwright — frontend;
- SQLite — persistence;
- Docker и Docker Compose — production image и deployment smoke;
- browser screenshots/responsive runs — визуальная проверка;
- web research и image generation — локальные изображения Planes и UI assets.

## Вклад разработчика и AI

Разработчик:

- выбрал концепцию лаборатории нестабильных порталов и сеттинг Magic/Omenpaths;
- определял и утверждал gameplay semantics, balance, UX и scope;
- отклонял лишние механики и неудачные варианты интерфейса;
- выбрал Go, WebSocket, TDD, ускоренный block delivery и deployment priority;
- предоставил визуальные референсы и проводил решающий ручной UI/deploy review;
- подготовил VPS, домен, nginx, HTTPS и серверные persistent directories.

AI:

- восстановил и поддерживал spec, roadmap, plans и traceability;
- переводил решения в domain invariants и deterministic tests;
- реализовал domain, simulation, persistence, REST, WebSocket, Tutorial, frontend,
  anonymous multi-lab sessions, assets и containerization;
- фиксировал RED/GREEN commits, разбирал failures и выполнял corrective passes;
- подготовил deployment runbook и помог диагностировать медленную Docker build.

Реализация была AI-assisted, но итоговые product decisions и acceptance оставались
за разработчиком.

## Ключевые запросы и повороты

1. Восстановить контекст из репозитория до кодирования и считать Final Spec
   единственным product/domain source of truth.
2. Реализовать Stages 3–8 последовательно через test → RED → minimal GREEN →
   verification → traceability → commit.
3. Провести сверку Blocks A и B перед продолжением.
4. Ускориться и реализовывать Blocks C, D и E целиком.
5. Определить Recommendation и Tutorial semantics до их реализации.
6. Уместить все семь Portal Slots на Dashboard без page scroll.
7. Заменить первоначальный интерфейс единой тёмной магической лабораторией.
8. Добавить anonymous persistent isolated laboratories вместо одной глобальной игры.
9. Развернуть один application container за уже настроенным host nginx.
10. После deploy исправить Tutorial navigation, visual hierarchy, favicon и Override.

## Мои ключевые решения

### Domain

- Plane — постоянный мир; Portal — уникальный instance открытия. Связь Plane 1:N Portal.
- Существует семь постоянных Portal Slots со стабильными позициями.
- Постоянный roster содержит 20 Observers.
- Lifecycle: AVAILABLE → OUTBOUND → EXPLORING → WAITING_RETURN → RETURNING →
  AVAILABLE; LOST является terminal.
- Transit занимает random 5–15 секунд, research — 20 секунд.
- Plane становится EXPLORED только после успешного Return.
- Первое перемещение навсегда задаёт Portal direction OUTBOUND или INBOUND.
- В одном Portal одновременно перемещается максимум один Observer.
- Creatures занимают corridor, проходят по одному за две секунды и блокируют transit.
  Статистика смертей намеренно исключена.
- Portal Energy вычисляется из baseline, времени и индивидуального decay; значение
  не записывается в БД каждую секунду.
- Natural/manual close даёт CLOSED; Energy depletion/hidden instability —
  COLLAPSED.
- Stabilize стоит 20 Lab Energy, меняет UNSTABLE на STABLE и добавляет 15 Portal
  Energy; выше overcharge boundary команда запрещена.
- SEND и RECALL стоят 0, CLOSE — 5, EXTRACTION — 30.
- Collapse обнуляет Lab Energy и включает Leyline Override на 20 секунд.
  Во время Override CLOSE и STABILIZE бесплатны.
- Extraction требует подходящий Plane хотя бы с одним ожидающим Observer. После
  пяти секунд sync автоматически возвращается longest-waiting всё ещё подходящий
  Observer. Если он ушёл через другой Portal, выбирается следующий; если никого
  не осталось, автоматического Return нет, остальной lifecycle не меняется.
- Risk определяется меньшим lifetime от scheduled time и Portal Energy плюс
  instability penalty согласно Final Spec.
- Recommendation в первую очередь защищает Observer movement и Lab resources,
  учитывая explored state Plane и присутствие Observers.

### Runtime и persistence

- Backend: Go, chi, WebSocket и SQLite.
- Clock и Random внедряются как зависимости для тестов без sleep и flaky random.
- LabManager владеет serialized resolve-first mutations и simulation loop.
- Перед time-sensitive command или derived read lifecycle разрешается под тем же
  lock; terminal transition выигрывает у опоздавшей команды.
- Основной TDD protocol:

```text
requirement → test → RED → minimal implementation → GREEN → verification
```

- REST обслуживает reads/commands; WebSocket отправляет initial snapshot,
  periodic updates и post-action updates.
- Одна SQLite schema обслуживает все лаборатории через lab_id. Отдельных таблиц
  или БД для каждой лаборатории нет.
- Anonymous browser session живёт 30 дней. Reload продолжает ту же лабораторию;
  expired/cleared session создаёт новую изолированную лабораторию.
- Session identity остаётся server-side и не попадает в gameplay DTO.

### Frontend и deployment

- Frontend: React + TypeScript, единый arcane laboratory shell.
- Семь равных Slots всегда видны как центрированная схема 4+3 без Dashboard scroll,
  включая compact phone layout.
- Occupied card целиком открывает Portal Details с keyboard support; вложенные
  commands остаются отдельными controls.
- Empty-slot actions disabled и не нажимаются.
- Portal Details умещается в viewport; только History имеет bounded internal scroll.
- Risk и Recommendation имеют semantic colors.
- Observer Life показывает 20 markers: available — green, lost — red, остальные
  живые вне Lab — dark.
- Tutorial продвигается по authoritative events. Завершённый короткий step остаётся
  виден достаточно долго; существующий Forward управляет переходом из Details.
- Обычные bottom-right notices удалены. Connection status виден только при
  reconnect/error и использует тематический English copy.
- Для всех 85 Planes используются локальные text-free artworks.
- Production — один non-root multi-stage image. Один Go process обслуживает SPA,
  REST, WebSocket и health.
- Public path: existing nginx/HTTPS → 127.0.0.1:8080; SQLite bind mount:
  /opt/omenpath/data.

## История стадий

Таблица служит индексом и не повторяет каждый checkpoint. Указана representative
RED/GREEN пара; промежуточные commits сохранены в Git.
Stage 20 отдельно завершил Tutorial UI; ниже он сохранён как самостоятельная строка.

| Stage | Результат | Evidence | Status |
|---|---|---|---|
| 0 | executable requirements и traceability catalog | a7e80a7 | GREEN |
| 1 | Go foundation, Clock/Random fakes, первый lifecycle | RED 0277adc → GREEN 1d77dcc | GREEN |
| 2 | Portal lifecycle, Energy, stability, creatures, risk, Slots | RED 04c990c → GREEN 9b06e78; close 4b0274d | GREEN |
| 3 | Observer lifecycle, Return, LOST, ordering | RED 71562a0 → GREEN 5e1391f; close 69c34a2 | GREEN |
| 4 | SEND/RECALL, direction, restrictions, atomic flow | RED e67667b → GREEN f883f23; close 6b1728a | GREEN |
| 5 | Lab Energy и transactional action costs | RED c79ee53 → GREEN 0c9e018; close a6167c0 | GREEN |
| 6 | collapse orchestration и Leyline Override | RED d7340f8 → GREEN cb4a3ca; close a6b5643 | GREEN |
| 7 | Extraction selection, sync и automatic Return | RED a566111 → GREEN c427b3d; close e6ea0e2 | GREEN |
| 8 | scheduler, ticks, максимум 7 open Portals | RED 6c6f494 → GREEN 1e58efe; corrective 48bb89a | GREEN |
| 9 | events, Portal history, deterministic event stream | RED d9ec172 → GREEN d2fe178; close 5131cbe | GREEN |
| 10 | SQLite migrations, atomic persistence, restart recovery | RED 573716c → GREEN bf6d132; close bb2969f | GREEN |
| 11 | LabManager ownership, locking, loop, cancellation | RED 0707336 → GREEN 38cf7f8; close cd651d7 | GREEN |
| 12 | REST, errors и Recommendation Engine | RED 3574dc8 → GREEN e4d5144; close 5c3a97f | GREEN |
| 13 | authoritative WebSocket snapshots и lifecycle | RED 3a2b07f → GREEN 99aa484; close 6adc8cf | GREEN |
| 14 | deterministic Tutorial engine и Live handoff | RED 5a95e37 → GREEN a569eb6; audit e0d08d5 | GREEN |
| 15 | React workspace, routes, clients, state, artwork | RED 7ed451f → GREEN f2dfb3e; close 6026f39 | GREEN |
| 16 | responsive seven-slot Dashboard и quick actions | RED f7ded73 → GREEN ae7e872; close bcffd59 | GREEN |
| 17 | Portal Details, diagnostics, actions, history | RED 393e9c7 → GREEN 9f75dfc; close ac7c835 | GREEN |
| 18 | Extraction chooser, confirmations, errors | RED ca3ddb6 → GREEN 3b1b72b; close 97369a3 | GREEN |
| 19 | realtime Event Log, filters, Tutorial signal | RED 78229d1 → GREEN ea87a2c; close 6d5dd1c | GREEN |
| 20 | contextual Tutorial UI, retry, Live handoff | RED ff48bbd → GREEN e0e9e9a; close e069768 | GREEN |
| 21 | single-source AI Worklog UI и integration | RED 8f92825 → GREEN 997b00b; close e94d1fa | GREEN |
| 22 | production health endpoint и SPA serving | dcd617a | GREEN |
| 23 | production image, Compose, env, persistent data | build RED → GREEN 31b03ca | GREEN |
| 24 | existing-nginx deployment runbook | 42b3086 | GREEN |
| 25 | container health, REST/WS и restart persistence smoke | 2fc78ee | GREEN в reduced scope |

## Крупные corrective passes

### Stage 2 и Blocks A–B

- Derived state terminal Portal заморожен на ClosedAt.
- Исправлены Energy/instability tie и chronological exploration.
- Покрыты late Extraction sync, scheduler invariants и canonical aggregates.
- Representative sequences: 04c990c → 9b06e78 и 4a192a4 → 48bb89a.

### Block C audit

- Research completion events сделаны независимыми от tick cadence.
- Усилены SQLite timestamps, pragmas, read consistency и failure visibility.
- Покрыты stale ticks, cancellation, random rollback и shutdown LabManager.
- Исправлены REST error mapping и WebSocket handoff/shutdown.
- Покрыты Tutorial retry, replacement SEND и atomic Live handoff.
- Финальная сверка: e0d08d5.

### Block D corrective gate

- Roster изменён с 10 на 20.
- Глобальный runtime заменён anonymous 30-day multi-lab isolation.
- Все persistence paths scoped по lab_id.
- REST и WebSocket привязаны к нужному laboratory runtime.
- Добавлена authoritative Observer transit projection.
- Переработаны Dashboard, Details, Event Log, Help, Tutorial и motion.
- Все 85 Plane artworks заменены локальными text-free assets.
- Deep-link session bootstrap/retry сериализован до route REST/WS.
- Representative sequence: c235c95 → afb1e76; final evidence b752182 и caf1d8b.

### Post-deployment UI

- Первая UI-реализация была отклонена как не соответствующая референсу.
  Commits bf9c03b–91fcc5a пересобрали её в единый arcane shell с 4+3 board,
  compact Details, semantic values и whole-card navigation.
- Observer markers, Tutorial Details и Override исправлены парами
  e4e844c → 359c16f, 5ca3869 → 5ed1d5d, 79ff681 → ba407fd.
- Favicon добавлен в 3ae1377.
- Отвергнутое CSS-светошоу Override заменено branching canvas lightning
  в 27a9529.

## Где AI ошибался

- Первый frontend-only proposal недооценил domain, realtime и persistence.
- До магического сеттинга рассматривались Stargate и sci-fi visual direction.
- Первая Observer model содержала слишком много historical timestamps.
- Fixed 10-second transit был заменён на random 5–15 seconds.
- Creature deaths не давали полезного gameplay и были удалены.
- Предложение автоматически вернуть всех Observers через Extraction отклонено.
- CLOSED и COLLAPSED пришлось снова разделить после ошибочного объединения.
- Sorted Portal list конфликтовал с семью стабильными Slots.
- Recommendation и Tutorial сначала были определены недостаточно точно.
- Первый frontend выглядел как sci-fi admin panel и дробил страницу.
- Часть Plane assets была placeholder или содержала card text.
- Tutorial преждевременно уводил пользователя со страницы Details.
- Observer summary дублировал Observer Life и занимал место.
- Первая усиленная Override animation оказалась светошоу, а не молниями.

Эти пункты сохранены намеренно: они показывают реальные corrections, а не
искусственно идеальную историю.

## Ручные правки и решения

Главные developer reviews:

- frontend-only → полноценный realtime backend;
- FastAPI → Go;
- Magic/Omenpath setting и dataset из 85 Planes;
- Extraction fallback selection и close-during-transit semantics;
- приоритеты Recommendation и пошаговый Tutorial;
- все семь Portals без Dashboard scroll;
- единая Magic-inspired композиция по пользовательскому изображению;
- одна server game → anonymous persistent isolated laboratories;
- один application container за existing nginx;
- полная переделка первого deployed UI и Override animation.

Это были не косметические правки, а решения, определившие итоговый продукт.

## Verification

Выполнено:

- Blocks A–B: gofmt, Go vet/build/test/race и traceability reconciliation.
- Block C: full ordinary/race suite и independent final audit.
- Block D: Go vet/build/test/race, frontend format/lint/typecheck/test/build,
  85/85 artwork audit, offline asset preparation и browser suite:
  47 PASS, 5 явных project-specific skips.
- Reference-faithful UI: frontend format/lint/typecheck, 192 tests, production
  build, Go vet/build/test, desktop/phone render review и image smoke.
- Dashboard/Tutorial corrective до 15c3710: frontend format/lint/typecheck,
  37 files / 194 tests, production build, gofmt и Go vet/build/test.
- Block E reduced gate: production build, loopback Compose config, healthy
  container, health, SPA, REST, WebSocket 101 и session persistence после recreate.
- На VPS пользователь подтвердил запуск на 127.0.0.1:8080 и ответ
  GET /health: {"status":"ok"}.

Некоторые первые local runs были RED только потому, что sandbox запрещал localhost
test listeners. Те же suites прошли с разрешённым listener; это environment RED,
а не скрытый product failure.

Canvas-lightning replacement 27a9529 по прямому указанию разработчика выполнен
без automated tests и принят визуально. Текущее сокращение Worklog не меняет код.

## Deployment snapshot

- URL: https://omenpath.duckdns.org/
- internal port: 8080
- host binding: 127.0.0.1:8080
- health: /health
- WebSocket: /ws/lab
- persistent host path: /opt/omenpath/data
- restart policy: unless-stopped
- public boundary: existing host nginx + HTTPS

Проект не менял VPS users, firewall, DNS automation, Certbot, host nginx,
CI/CD, Kubernetes или monitoring.

## Будущие улучшения и отложенная работа

Не отмечены как GREEN:

- balance simulation на 10 000–100 000 Portals;
- новый exhaustive race/Playwright matrix после последних presentation-only changes;
- расширенный external manual и cross-browser gameplay QA;
- optional English/Russian language switch;
- финальный whole-repository consistency audit после всех post-deployment corrections.

Будущие записи должны быть краткими: scope, решение, evidence, verification и
явные omissions. Не нужно возвращать пересказ каждой команды, уже видимой в Git.
