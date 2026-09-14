# Triage labels

Use this mapping when a skill names a triage role. The local tracker records the role in a ticket's `Status:` line.

| Canonical role | Label in this tracker | Meaning |
| --- | --- | --- |
| `needs-triage` | `needs-triage` | A maintainer needs to evaluate the issue |
| `needs-info` | `needs-info` | Waiting for more information from the reporter |
| `ready-for-agent` | `ready-for-agent` | Fully specified and ready for an agent to implement |
| `ready-for-human` | `ready-for-human` | Requires human implementation |
| `wontfix` | `wontfix` | Will not be implemented |

Edit the tracker-label column if the repository adopts another vocabulary. See [Issue tracker](issue-tracker.md) for storage conventions and wayfinding execution states.
