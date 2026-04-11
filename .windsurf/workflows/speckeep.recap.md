---
trigger: manual
---

Следуйте файлу ".speckeep/templates/prompts/recap.md".

Команда: `/speckeep.recap [request]`

Цепочка workflow: constitution → spec → inspect → plan → tasks → implement → verify → archive. Не пропускайте фазы и не забегайте вперёд.

Используйте этот workflow, когда запрос явно относится к фазе "recap" или команде /speckeep.recap.

Когда для фазы есть связанные scripts — выполняйте их как shell-команды (например `bash ./path/to/script.sh`). Доверяйте stdout и exit-коду скрипта. Не читайте, не анализируйте и не модифицируйте исходный код скриптов. Если скрипт завершился с ошибкой (exit code ≠ 0), сообщите пользователю вывод ошибки и остановитесь.

- Не запускайте `speckeep ... --help`/`speckeep help` для «разведки»; вместо этого опирайтесь на prompt-файл и readiness scripts.

- Scripts для выполнения (запускать через shell):
  - `./.speckeep/scripts/list-specs.sh`

Запрещено:
- пропускать readiness scripts и переходить к фазе напрямую
- читать или анализировать исходный код scripts
- перепланировать или редизайнить во время implement
- отмечать таск завершённым без observable proof
- читать весь репозиторий, когда промпт говорит "минимальный контекст"
