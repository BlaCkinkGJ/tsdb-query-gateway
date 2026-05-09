# Implementation Plan: Architecture and Robustness Improvements

## Objective
Implement the architectural and stability improvements identified during the codebase investigation. This includes enforcing strict middleware configuration validation, handling Prometheus scalar/string types correctly during merging, and introducing basic de-duplication for vector results.

## Scope & Impact
- `prom-router/internal/middleware/`: Update factory signatures to return errors. Enforce `endpoint` requirement in AI middleware.
- `prom-router/cmd/prom-router/main.go`: Update initialization logic to handle middleware factory errors as fatal.
- `prom-router/internal/service/router.go`: Update `mergeResults` to handle scalar/string types (return first successful) and de-duplicate vector results based on metric labels.

## Implementation Steps

### Phase 1: Middleware Configuration Strictness
1. **Update `middleware.go`**:
   - Change the `Factory` signature to `func(...) (Middleware, error)`.
   - Update `Chain` or related functions if necessary.
2. **Update `ai.go` (and other middlewares)**:
   - Modify the factory function to return an error if `endpoint` is missing, instead of just logging a warning.
3. **Update `main.go`**:
   - Handle the `error` returned by `factory()`. If `err != nil`, call `log.Fatalf` to prevent the server from starting with an invalid state.

### Phase 2: Result Merging Robustness
1. **Update `router.go` (`mergeResults`)**:
   - **Scalar/String Types**: Add a check. If `mergedResultType` is `scalar` or `string`, return the first successful result instead of attempting to merge arrays, as these types are fixed-size tuples.
   - **Vector De-duplication**: When merging `vector` or `matrix` types, implement a mechanism (e.g., hashing the labels) to ensure that identical time series from different downstream Prometheuses are not duplicated in the final array.

## Verification & Testing
- Run all existing unit tests in `prom-router`.
- Add tests to `router_test.go` to specifically test scalar merging (should return only one result) and vector deduplication.
- Run `go build ./...` to ensure there are no compilation errors.