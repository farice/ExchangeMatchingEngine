- Get locally generated uid for both redis and postgres
- Hook up redis in model.go
- Soft-delete in redis on initial deletes
- Hard-delete in redis after confirmation from database
  - Figure out how messages of deletion should come back, if at all.
- Write matching logic
- Test speed of SQL query flushing. May need to explore options for dispatch.

**Bugs:** 
- createOrUpdateSymbol() only re-creates symbols; does not balance to existing symbols.
