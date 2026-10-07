# Независимая проверка корректности композиции

Проверено: `.cursor/docs/task1.md`, optional module `integrations/recipes`, consumer script/workflow, stream/accounting/capture/export contracts. Implementation не изменялся. Проверка выполнена до релиза и closeout.

## Результат

Повторная независимая проверка: оба найденных P2 исправлены. Открытых найденных дефектов нет. Полный optional suite и исходные независимые reproducer tests прошли с `-race -count=1`: `/tmp/evaly-correctness-final.log`, exit status 0. Новые регрессионные тесты также прошли. Ниже сохранены исходные находки и подтверждение исправления.

### Закрыт P2 — invalid sidecar ломал changed-content conflict

- Место: `integrations/recipes/export.go:26`, `integrations/recipes/export.go:209`–`213`.
- `LocalRegistry.Deliver` валидирует только SDK Evaluation, а не полный EvaluationRecord. У sidecar Metric может быть Value=NaN. `json.Marshal` тогда возвращает ошибку, но `identity` игнорирует её и хеширует пустые bytes.
- Воспроизведение: создать registry с `genai.NoopEvaluationRecorder`; доставить record с ObservationID="same", Status=EvaluationSucceeded, Outcome="pass", Metric.Value=NaN; повторить с Outcome="fail". Оба вызова возвращают nil; изменённое содержимое с тем же ID принимается как duplicate. Первый невалидный record также не отклоняется до side effect.
- Проверка: `/tmp/evaly-correctness-review/review_test.go`, `TestReviewInvalidSidecarConflict`, запуск `GOPATH=/tmp/evaly-gopath GOCACHE=/tmp/evaly-go-build GOWORK=off go -C /tmp/evaly-correctness-review test -race -count=1 -run TestReview -v` — FAIL с сообщением `invalid sidecar accepted and same-ID changed-content accepted without conflict`.
- Действие: полная валидация sidecar либо как минимум обязательная обработка marshal error до Recorder.Record и обновления registry. Invalid JSON value не должен иметь digest успешного содержимого. Добавить регрессионный AAA test для NaN/Inf в sidecar и changed-content conflict.

### Закрыт P2 — privacy projection подменяла привязку результата

- Место: `integrations/recipes/export.go:167`, `integrations/recipes/export.go:186`–`189`.
- `sameMeasurement` защищает ObservationID и score, но не association/provenance/artifact references. При этом публичный README разрешает их redaction, а нормативный контракт запрещает identity rewrites. Замена значений на другие допустимые identity не является redaction.
- Воспроизведение: EvaluationSink.Project присваивает `Association.TrialID="unrelated-trial"` и `Provenance.GraderReference="unrelated-grader"`, сохраняя ObservationID и measurement. Run → envelope → Export возвращает delivered и передаёт обе записи sink. Стабильный ID исходного результата теперь прикреплён к чужому trial/grader.
- Проверка: тот же standalone reproducer, `TestReviewProjectionChangesAssociation` — FAIL: `identity rewriting accepted: state=delivered deliveries=2`.
- Действие: разрешить для linkage/reference fields только исходное значение или документированное удаление; проверять целостность полностью редактируемого association. Чужое непустое значение отклонять с ErrConflict до первой доставки. Добавить AAA tests на trial/grader/artifact substitution и разрешённое удаление.

## Подтверждение исправлений

1. `validateRecord` теперь проверяет весь EvaluationRecord: вид результата, статус, соответствие DTO и sidecar, допустимые metric value/range/scale/direction. Registry делает эту проверку до SDK side effect. Marshal failure возвращает пустой digest, который registry явно отклоняет. `TestRegistryRejectsInvalidSidecarBeforeDedup` проверяет ErrInvalid и отсутствие spans; независимый `TestReviewInvalidSidecarConflict` теперь PASS.
2. `sameMeasurement` проверяет association, provenance, artifact reference и trace. Непустые чужие ссылки отклоняются; исходное значение либо удаление допускаются, а SDK validation сохраняет целостность association tuple. `TestExportRejectsForeignAssociationAndProvenance` проходит для trial/grader/target/artifact/trace, гарантируя conflict и ноль deliveries. Независимый `TestReviewProjectionChangesAssociation` теперь PASS. Privacy fixture удаления ссылки проходит.

## Проверенные границы

Core не импортирует optional SDK module. Target синхронно потребляет реальный SDK stream и сохраняет partial receipt; lazy nil handle не принимается. Capture консервативно сохраняет incomplete/sampled flags; unknown conversion не выдаёт known zero. Отдельные assertion/metric/status IDs включают artifact revision, trial, full grader identity и policy. Сериализация delivery/dedup через mutex проходит существующие race fixtures. SDK evaluation реально создаёт и завершает spans; durable collector acknowledgment не обещан.

Логи `/tmp/evaly-consumer-published.log` и `/tmp/evaly-consumer-source.log` содержат PASS всех существующих fixtures. Лог `/tmp/evaly-validate.log` просмотрен; окончательный exit status root validation подтверждает исполняющий агент. Повторный независимый полный suite на финальном export.go закрывает оба дефекта. Финальные consumer matrix подтверждает исполняющий агент по их завершению.

Отсутствие дополнительных findings не означает доказанное отсутствие любых ошибок. Адресная повторная проверка выполнена; успешный релиз и внешние closeout действия проверяет исполняющий агент отдельно.
