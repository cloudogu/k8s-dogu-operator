# Generating mocks with mockery

Auto-generate mock files for your interfaces (located in controllers/mocks) with mockery:

- Install mockery, e.g. via `sudo apt install mockery`
- Switch to controllers directory, e.g. via `cd controllers`
- Generate mock files for your interface, e.g. via `mockery --name ImageRegistry`

For interfaces in the `internal` package exist a make target `make mocks`.