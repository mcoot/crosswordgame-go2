# Agent instructions

## Notes

- `./docs/design` contains human-authored high level design docs
    - These should be taken as the base architecture. They are not infallible, but generally should be respected unless a change to them is specifically approved by the human architect
    - Ensure the guidelines in design are followed, especially in regards to tech stack, code structure and testing strategy
- `./docs/specs` contains specs for implementation tasks, following the `./docs/specs/spec-000-template.md` template format and naming scheme
    - An agent should create a new spec when starting a major task. The spec can then be used for human architect approval, and for another agent to continue where one left off.
    - Agents should refer to active and in progress specs when needed to understand prior changes or parts of the architecture
- Always use `task` when trying to run standard commands like building, testing, linting etc.
    - If we have a new feature which should be come a standard command, we should add it to the taskfile
    - Always use standard modes of interaction. For example, using our cli tool (via task), rather than curl

## Development process

We focus on high quality and the code standards in our code design document (`./docs/design/code-design-tech-stack.md`).
