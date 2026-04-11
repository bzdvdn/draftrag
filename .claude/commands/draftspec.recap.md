---
description: Project-level overview of all active features and their current phase
argument-hint: [request]
---

Следуйте файлу ".draftspec/templates/prompts/recap.md".

Команда: `/draftspec.recap [request]`

Цепочка workflow: constitution → spec → inspect → plan → tasks → implement → verify → archive. Не пропускайте фазы и не забегайте вперёд.

Аргументы пользователя:
{{arguments}}

Требования:
- сначала прочитайте .draftspec/constitution.md, если это требуется prompt-файлом
- используйте только минимально нужный контекст репозитория
- Когда для фазы есть связанные scripts — выполняйте их как shell-команды (например `bash ./path/to/script.sh`). Доверяйте stdout и exit-коду скрипта. Не читайте, не анализируйте и не модифицируйте исходный код скриптов. Если скрипт завершился с ошибкой (exit code ≠ 0), сообщите пользователю вывод ошибки и остановитесь.
- Scripts для выполнения (запускать через shell):
  - `./.draftspec/scripts/list-specs.sh`
- Не запускайте `draftspec ... --help`/`draftspec help` для «разведки»; вместо этого опирайтесь на prompt-файл и readiness scripts.
- обновляйте только релевантные артефакты и кратко сообщайте об итогах и блокерах

Запрещено:
- пропускать readiness scripts и переходить к фазе напрямую
- читать или анализировать исходный код scripts
- перепланировать или редизайнить во время implement
- отмечать таск завершённым без observable proof
- читать весь репозиторий, когда промпт говорит "минимальный контекст"
