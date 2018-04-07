# Danger log

- If we're performing the symbol (create) command and one of the accounts does not exist (but the others do), do we roll back the additions of the symbol to the other accounts?
  - Drew's response: do something reasonable, not specified
  - My suggestion: add to all of the accounts possible, return one error that describes which accounts the symbol couldn't be added to

- Idempotency
  - There are several locations where it's essential that we keep the FSM design in mind. For example, buying, selling, or cancelling.
  - There are many points with which things can fail and we need to retry and ensure that this retrying has no unintended consequences
  - E.g. Account for double refunding (for cancels): Use atomic transactions that have unique transaction identifiers. Hence, if we receive multiple requests to cancel the same transaction ID, we'll ignore all but one.

- Atomicity
  - Some operations require several steps to complete. And if any step fails, the entire process needs to be rolled back. For example, when we cancel an order we need to both remove the order and refund the user. If one fails, we must roll back to ensure we have idempotency i.e. the user can retry without unintended consequences

- Concurrent buying/selling
  - Mutual exclusion is essential. We can't match two open orders to the same waiting order. Hence, we use a mutex anytime there is a search for a potential order match.

- Persistence correctness in crash
  - We were not able to implement fail-safes to ensure correctness in the event of a crash. This entails not just lost data, but cache inconsistency on restart. This is due to the fact that redis persists its cache to an "append-only file" which allows the cache to be restored in its existing state. In contrast, our write buffer for the postgres database has no such safeguard. As a result, data could be written to the cache, persists through a crash, but be lost for the underlying data store. In this event the cache would be inconsistent
