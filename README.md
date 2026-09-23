# Task Tracker CLI

Консольный трекер задач на Go. Использует только стандартную библиотеку,
задачи хранит в JSON-файле в текущей директории.

Решение проекта [Task Tracker](https://roadmap.sh/projects/task-tracker) с roadmap.sh.

## Сборка

Требуется Go 1.27.1 и новее.

```
go build -o task-cli .
```

## Использование

```
task-cli <команда> [аргументы]
```

| Команда | Описание |
| --- | --- |
| `add <описание>` | Добавить задачу со статусом `todo` |
| `update <id> <описание>` | Изменить описание задачи |
| `delete <id>` | Удалить задачу |
| `mark-in-progress <id>` | Установить статус `in-progress` |
| `mark-done <id>` | Установить статус `done` |
| `list` | Показать все задачи |
| `list <статус>` | Показать задачи с указанным статусом (`todo`, `in-progress`, `done`) |

### Пример

```
$ task-cli add "Buy groceries"
Task added successfully (ID: 1)

$ task-cli add "Cook dinner"
Task added successfully (ID: 2)

$ task-cli list
1    todo         Buy groceries
2    todo         Cook dinner

$ task-cli mark-in-progress 1
Task status updated successfully (ID: 1)

$ task-cli list in-progress
1    in-progress  Buy groceries

$ task-cli delete 2
Task deleted successfully (ID: 2)
```

Ошибки выводятся в stderr, программа завершается с кодом 1.

## Хранение данных

Задачи хранятся в файле `tasks.json` в той директории, откуда запущена команда.
Файл создаётся при первой записи; отсутствующий или пустой файл считается
пустым списком задач.

```json
[
  {
    "id": 1,
    "description": "Buy groceries",
    "status": "todo",
    "createdAt": "2026-09-22T21:43:15.1006313+03:00",
    "updatedAt": "2026-09-22T21:43:15.2672437+03:00"
  }
]
```
