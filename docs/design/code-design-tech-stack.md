# Code design and Tech stack

This document is used to define the code style, and core tech stack used.

We do not need to list every small library here.

## Principles

- We prefer simpler to more complex
- Our code structure is aimed at refactorability/writing for change
- We use testing and test-driven development practices to build certainty in our code
- We take care to keep our dev / iteration loop simple and fast
- We abide by the principles of Hypermedia and progressive enhancement in our web UX
    - We avoid making a thick clientside application
- We are careful in adding dependencies only when necessary, but do not re-implement complex logic when a well-established library exists

## Core tech stack choices

- The game is written in Golang, using the most recent available version (at time of writing, 1.25)
- We use `taskfile` for defining build tasks
- We will make use of github actions for CI/CD, being judicious to keep within free limits
- In specs we use Mermaid diagrams when diagramming
- We use `testify` for testing. We do NOT use a specific mocking library, we will use interfaces instead.
- The game is exposed ultimately as an HTTP server
- The game's logic is implemented with an internal (code) API layer
- We expose two HTTP-based APIs on top of this:
- The first such API is a JSON API that can be interacted with by an automated system for testing, or a CLI for manual usage
- Secondly however, we provide the game's web interface as an HTML hypermedia API with server-side templating, using HTMX to provide progressive enhancement for a modern-feeling UX
    - We use [Templ](https://github.com/a-h/templ) for server-side template rendering
    - We use SSE (and the HTMX plugin for SSE) to provide live game state updates
    - We can use `gorilla` for mux and session handling
- We have internal abstractions over the data storage layer and initially provide an in-memory implementation of data storage
    - However we should eventually make this compatible with other backends such as a Redis or Postgres for persisting the storage. We should make data modelling decisions so as to make changing the underlying data layer in future easy.
- We use the `slog` standard logging format
- We can use `viper` to supply app configuration via env var or file (for dynamic reload)

## Code structure

Although we don't have a true DI framework, we use Dependency Injection principals. Most code should be defined in terms of a DI component or service, represented by a struct which contains all dependencies with a constructor to inject them. We will use a factory containing the wiring up of all dependencies into the overall application.

All external dependencies to the system, such as calls to external systems, the filesystem, time, loggers, etc. should be wrapped in a dependency interface we define, and we will mock these for unit tests with an implementation of the interface.

App configuration loading should also be indirected by an interface.

We are very careful in defining our _boundaries_ in terms of test bubbles. We generally want to create large test bubbles, and in unit tests we only ever mock at the boundary of that bubble - for instance, external dependencies. We will need to make decisions about where to draw internal test bubbles (and hence where we mock other components); we prefer fewer, larger test bubbles to ensure that the unit tests give a high degree of certainty about the whole system, but we must balance that with practicality (e.g. avoiding combinatorial explosion of test cases). We should carve out test bubbles along clear and natural lines (e.g. perhaps the data layer would form a natural test bubble), and treat the contracts between these test bubbles with care, since we will not have unit tests across them.

Our high level package structure looks like:
- `./cmd/<binary>/main.go` - main for a given binary
- `./data` - contains data such as accepted dictionary words
- `./internal` - code root for internal code
- `./internal/model` - data models
- `./internal/dependencies` - thin interfaces and clients for any external dependency
- `./internal/dependencies/mocks` - mocks of those client interfaces
- `./internal/factory` - DI wiring for the application
- `./internal/services` - individual _services_ (i.e. DI components) with logic; may be in sub-packages as needed. Where we draw test bubbles between certain services, we may need mocks (in sub-packages) for some services.

Note we will have one main binary for the server, but may find a need for others for tooling.

## Testing strategy

### Unit testing

For our core business logic in particular, we want unit testing to give us a high degree of certainty that our test bubbles are correct. Unit testing is the cheapest layer of testing and so should be the most comprehensive.

We use red-green test-driven development when implementing this business logic. We ensure that at the end of implementing a task, there is no way we could break or regress the logic without failing a test.

We write unit tests using `testify` and organise them into suites aligned to a logical service. We may have more than one suite for a service as needed where multiple suites would be clearer.

We create a test fixture struct associated to a service and have test wiring in addition to our real wiring factory, where the test wiring wires in mocks for external dependencies or services outside the current bubble.

We format our test suite structure, names and test names consistently, describing the conditions, actions and expected result.

We use arrange/act/assert format in tests.

### Integration testing

While our core implementation of business logic should be mostly covered by unit tests and that is preferable (since it is faster/lower complexity), we will need to have a strategy for testing our business logic in a more end-to-end fashion.

We should have a suite of end-to-end tests that runs through entire flows using the JSON API against a real server . We need to balance keeping the dev loop fast here, so we should not be exhaustive, but we should cover key normal and edge cases.

### Web interface flow testing

To test our HTML web interface, we will need to invest in full tests that run through the web flow. We should leave implementing this (and the web interface) until our business logic core is solid and tested via the JSON API.

Then we will need to choose the simplest and most pragmatic approach to automated web testing, to be decided at that time. We will again not be exhaustive with this testing, but we want to build confidence our web interface layer is correct.