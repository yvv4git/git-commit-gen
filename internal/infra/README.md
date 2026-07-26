# infra

The `infra` package is the application's Composition Root.

It initializes all external dependencies: logger, HTTP clients, LLM clients, etc.

Commands and `main` do not wire dependencies manually. They call constructors from this package instead. This keeps entry points clean and avoids duplicating initialization logic.
