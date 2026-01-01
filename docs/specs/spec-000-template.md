---
spec_id: "spec-{incrementing-spec-number}"
spec_name: "{Task or feature name}
status: "{PROPOSED | IN_PROGRESS | ACTIVE | DEPRECATED}"
---
# {spec_id} - {Task or feature name}

## Overview

A concise overview of the change, including what is being done and why.

## Relevant context

Any context relevant to executing the task. This could be:
- code, doc or other spec references
- Notes on what is and is not in scope for this change/spec

Do not make this section overly long just for completeness, focus on key information.

## Task implementation strategy

The strategy for implementation including a numbered list of the high-level tasks to be done.

The numbered list should represent logical chunks, such that an architect can instruct the agent to work autonomously on a task in the list.

## Status details

In addition to the overall status at the top, we can store additional notes here. When a spec is IN_PROGRESS we should store the status of our strategy (e.g. which task we're up to).