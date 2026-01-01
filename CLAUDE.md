# Agent instructions

- `./docs/design` contains human-authored high level design docs
    - These should be taken as the base architecture. They are not infallible, but generally should be respected unless a change to them is specifically approved by the human architect
- `./docs/specs` contains specs for implementation tasks, following the `./docs/specs/spe-000-template.md` template format and naming scheme
    - An agent should create a new spec when starting a major task. The spec can then be used for human architect approval, and for another agent to continue where one left off.
    - Agents should refer to active and in progress specs when needed to understand prior changes or parts of the architecture