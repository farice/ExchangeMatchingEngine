# Danger log

- If we're performing the symbol (create) command and one of the accounts does not exist (but the others do), do we roll back the additions of the symbol to the other accounts?
  - Drew's response: do something reasonable, not specified
  - My suggestion: add to all of the accounts possible, return one error that describes which accounts the symbol couldn't be added to
