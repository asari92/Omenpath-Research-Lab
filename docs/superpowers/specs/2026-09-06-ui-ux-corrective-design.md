# Omenpath Research Lab — UI/UX corrective design

Дата: 2026-09-06
Статус: согласовано пользователем

## 1. Цель

Пересобрать frontend как цельную магическую исследовательскую лабораторию, исправить обнаруженные UX-дефекты Block D и добавить изоляцию игр разных посетителей одного server deployment.

Документ не заменяет `00_FINAL_SPEC_v5.md`. До реализации новые product-решения из этого дизайна должны быть внесены в Final Spec, requirements и traceability. Остальные gameplay semantics завершённых stages не меняются.

## 2. Подтверждённые product-изменения

1. Начальный и максимальный roster увеличивается с 10 до 20 Observers.
2. Один server deployment обслуживает несколько изолированных лабораторий.
3. Первый посетитель автоматически получает анонимную игровую сессию на 30 дней без регистрации.
4. Срок сессии скользящий: успешный REST request или активное WebSocket connection продлевает его до 30 дней. После истечения создаётся новая лаборатория, старая удаляется фоновой очисткой.
5. Один browser profile и его вкладки используют одну лабораторию. Другой browser/profile/device получает отдельную лабораторию.
6. Добавляется Help с лором, устройством лаборатории и описанием игровых механик.
7. RU/EN localization не входит в обязательный corrective pass и выполняется только при оставшемся времени. Текущий обязательный UI остаётся английским.

## 3. Визуальное направление

Интерфейс — единая dark-fantasy сцена магической обсерватории, вдохновлённая визуальной грамматикой fantasy trading-card games, но без копирования логотипов, mana symbols, card frames или иных фирменных материалов Magic: The Gathering.

Вся страница использует одну поверхность: тёмный камень, чернёная бронза, потёртая кожа, дымчатое стекло, тонкие ley-line схемы и умеренные цветные магические блики. Левая зона не получает отдельный фон, вертикальную стену или самостоятельную рамку. Декор окружает весь viewport и направляет внимание к изображениям миров.

Основные правила:

- минимум вложенных прямоугольных рамок;
- рамка и орнамент не конкурируют с Portal art;
- декоративные элементы продолжаются через весь экран;
- состояния читаются не только по цвету, но также по тексту, форме и иконке;
- текст остаётся контрастным и разборчивым;
- все motion effects имеют reduced-motion вариант.

Официальные материалы, использованные как принципиальные референсы:

- https://magic.wizards.com/en/news/making-magic/frames-reference-2003-01-27
- https://magic.wizards.com/en/news/announcements/dominaria-frame-template-and-rules-changes-2018-03-21

## 4. Глобальная оболочка

Одна декоративная рамка окружает всё приложение. Левая layout-зона содержит сверху brand и показатели лаборатории, снизу — навигацию и глобальные действия. Между ней и content area нет отдельного фона или тяжёлого разделителя.

Порядок навигации:

1. Dashboard;
2. Event Log;
3. Help;
4. Open Extraction и Artwork Credits как глобальные действия;
5. AI Workflow — последний пункт.

Состояние realtime отображается тематически:

- green: `Planar link stable`;
- amber: `Planar paths unstable`;
- red: `Disconnected from the planes`.

При нормальном состоянии индикатор компактный. Reconnecting/error становятся заметными и временно блокируют команды, но последний корректный authoritative snapshot остаётся видимым.

## 5. Dashboard

Dashboard целиком помещается в `100dvh` без прокрутки страницы или board.

Desktop board строится на восьми внутренних колонках:

- верхний ряд: Slots 1–4 занимают пары `1–2`, `3–4`, `5–6`, `7–8`;
- нижний ряд: Slots 5–7 занимают пары `2–3`, `4–5`, `6–7`.

Так нижняя тройка строго центрируется. Все семь Slots имеют одинаковую ширину и высоту независимо от наличия Portal. На узком экране используется `2 + 2 + 2 + 1`; все Slots остаются видимыми без прокрутки.

Каждый occupied Slot показывает уже разрешённую Final Spec информацию и дополнительно визуализирует активный Observer transit: Observer ID, направление и remaining time. Пустой Slot сохраняет те же размеры; все его действия являются настоящими disabled controls и не реагируют на pointer/keyboard activation.

UNSTABLE окрашивает фон всей карточки заметным красным слоем. Leyline Override меняет атмосферу всей страницы: проявляются ley lines, цветные всполохи и пульсация outer frame.

## 6. Portal Details

Portal Details также помещается в viewport без прокрутки страницы:

- крупный Portal находится по центру;
- Actions расположены непосредственно под Portal без отдельной framed section;
- слева размещаются Portal state, Energy, timer и Observer transit;
- справа — Destination, research, creatures, Risk и Recommendation;
- History находится компактной полосой снизу.

History сохраняется, потому что обязательна по Final Spec. В закрытом состоянии она показывает заголовок и количество событий. После раскрытия прокручивается только внутренний список History, а не вся страница.

Terminal Portal полностью обесцвечивается: destination image, portal ring, sparks и status decoration. Постоянная надпись `Refreshing…` удаляется; background refresh не создаёт визуального шума.

Risk treatment:

- LOW — green;
- MEDIUM — yellow;
- HIGH — orange;
- CRITICAL — red.

Recommendation содержит цвет, иконку, действие и короткое объяснение. Используются отдельные состояния safe, suggested, urgent и unavailable.

## 7. Tutorial

Tutorial — floating grimoire поверх Dashboard и не участвует в layout flow, поэтому не сдвигает контент. Он размещается так, чтобы не закрывать Portal action, требуемый текущим шагом.

`More context` удаляется; весь текст текущего шага виден сразу. Каждый экран содержит:

- что происходит в системе;
- что объясняется;
- что должен сделать игрок;
- условие завершения.

Выполнение ожидаемого действия немедленно переводит tutorial на следующий authoritative step. Если между UI renders сервер успел завершить промежуточный step, UI помещает его в presentation queue, показывает как выполненный в течение 7 секунд и затем автоматически догоняет текущий authoritative step.

`Back` открывает любой уже показанный step. `Forward` возвращает к текущему step; будущие невыполненные задания пропустить нельзя.

После terminal tutorial Portal сервер по-прежнему немедленно создаёт замену по действующим semantics, но UI сначала показывает двухсекундную exit animation, затем entrance animation нового Portal.

## 8. Event Log и Help

Event timestamp отображается как `HH:mm:ss-dd-MM-yyyy`. Далее отдельными строками идут event title и details. Event type обозначается цветом и иконкой. Event Log допускает внутреннюю прокрутку списка, потому что число событий не ограничено высотой viewport.

Help — самостоятельная страница-справочник со следующими разделами:

- Multiverse и задача Omenpath Research Lab;
- интерфейс лаборатории;
- Lab Energy и Portal Energy;
- lifecycle и terminal states Portal;
- Risk и instability;
- Observers, transit и LOST;
- research и creatures;
- Stabilize, Close, Send и Recall;
- Extraction;
- Leyline Override;
- Recommendations;
- Tutorial recap и glossary.

## 9. Action feedback и motion

Каждая команда проходит одинаковую визуальную последовательность:

1. немедленное pressed state;
2. pending indicator и блокировка повторной отправки;
3. success pulse либо error pulse;
4. содержательное toast-сообщение.

Portal entrance использует раскрывающийся ring и sparks. Terminal transition обесцвечивает image, гасит glow и стягивает Portal внутрь. Dashboard хранит только краткоживущий presentation ghost предыдущего accepted snapshot; gameplay state не подменяется и не изменяется оптимистически.

При `prefers-reduced-motion` длительные движения заменяются короткими opacity/state transitions.

## 10. World artwork

Проверяются все 85 изображений. Обязательной замене подлежат существующие generated fallbacks и изображения с card text/frame, включая `14-bloomburrow`, `21-duskmourn`, `32-hell`, `75-thunder-junction`.

Порядок выбора:

1. подходящий landscape/art crop без текста из проверяемого источника;
2. если подходящего материала нет — генерация по каноническому описанию Plane в согласованной painterly style;
3. локальная нормализация в WebP и обновление attribution/source manifest.

Runtime не зависит от внешних image URLs.

## 11. Multi-lab session architecture

Используется одна SQLite database и общая схема с `lab_id`, а не отдельные tables или database files на лабораторию.

Новые сущности:

- `labs(id, created_at, expires_at, last_active_at)`;
- `sessions(id, token_hash, lab_id, expires_at, last_seen_at)`.

`planes`, `portals`, `observers`, `events`, `lab_state` и `app_state` получают `lab_id`. Singleton rows становятся singleton-per-lab. Entity keys и foreign keys включают `lab_id`; ни один repository query не может читать или изменять tenant-owned rows без resolved `lab_id`.

HTTP middleware:

1. читает opaque session cookie;
2. хеширует token и ищет незавершённую session;
3. при отсутствии создаёт lab, bootstrap state и session;
4. помещает typed `lab_id` в request context;
5. обновляет sliding expiration не чаще заданного refresh interval;
6. возвращает `HttpOnly`, `SameSite=Lax`, production `Secure` cookie.

Raw token в database не хранится. Token должен иметь не менее 256 bits entropy. Production deployment требует HTTPS.

Engine/manager registry keyed by `lab_id` обеспечивает отдельную serialization boundary и update stream для каждой лаборатории. WebSocket handshake разрешает session через ту же cookie и подписывает client только на hub соответствующего `lab_id`. Broadcast между лабораториями запрещён архитектурно.

Фоновая cleanup job удаляет expired sessions и labs каскадно. Живое WebSocket connection защищает laboratory от cleanup; renewal выполняется с bounded interval, а не database write на каждый tick. Cleanup, session refresh и gameplay mutations должны иметь определённый transaction/locking order, чтобы избежать удаления активной лаборатории.

Migration присваивает существующим singleton данным специальный legacy `lab_id`; новые чистые базы сразу создаются в multi-lab schema.

## 12. Data contracts

Authoritative REST/WebSocket snapshot расширяется активными transit projections. Для каждого transit доступны как минимум:

- `observer_id`;
- `portal_id`;
- direction/phase;
- `started_at`;
- `completes_at`.

Remaining time вычисляется из authoritative timestamps и snapshot resolution time. Dashboard и Details используют один DTO/helper, чтобы не расходиться по статусам.

## 13. Проверка и TDD

Работа выполняется checkpoint-by-checkpoint: tests → RED → minimal implementation → GREEN. Проверки охватывают:

- migration и tenant scoping каждого repository operation;
- cookie creation, renewal, expiry и hashing;
- isolation REST actions, Events и WebSocket broadcasts между двумя labs;
- cleanup race с активной session;
- roster 20 и tutorial reset;
- exact desktop `4 + 3` и mobile `2 + 2 + 2 + 1`;
- отсутствие page scroll на Dashboard и Portal Details;
- equal empty/occupied Slot geometry;
- disabled empty actions;
- transit display;
- tutorial 7-second presentation queue и two-second replacement transition;
- terminal grayscale, Risk/Recommendation treatments;
- connection states, action pending/success/error feedback;
- Help navigation и AI Workflow last;
- Event timestamp/content order;
- artwork coverage всех 85 Planes;
- reduced motion.

Stage verification следует `AGENTS.md`: gofmt, vet, build, full tests; block boundary дополнительно race suite. Frontend lint, unit/integration, build и responsive E2E выполняются до завершения corrective pass. Worklog и traceability обновляются вместе с фактическим coverage.

## 14. Не входит в обязательный scope

- username/password registration;
- перенос одной игры между browser profiles/devices;
- social login;
- horizontal server scaling;
- обязательная RU/EN localization.
