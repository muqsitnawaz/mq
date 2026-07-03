test: add unit coverage for lib.Reduce

## What

Adds `lib/operators_test.go` with `TestReduce`, covering the generic
`Reduce[T, R any]` helper in `lib/operators.go`. This function had no
direct test coverage — a repo-wide grep of `lib/*_test.go` found tests
for its sibling operators (`Filter`, `Map`, `Take`, `Unique`, etc.) but
none exercising `Reduce`.

## Why

`Reduce` is a small, well-defined, pure generic function used to fold a
slice into a single accumulated value. It's simple enough to test in
isolation without fixtures, but its behavior (especially the empty-slice
edge case, where it must return the initial value untouched) isn't
obviously correct from a glance and is exactly the kind of thing that
can silently break in a refactor.

## Test cases

- Numeric accumulation: summing `[1, 2, 3, 4]` with `0` as the initial
  value yields `10`.
- String accumulation: concatenating `["a", "b", "c"]` yields `"abc"`,
  confirming the reducer works across different `T`/`R` type params.
- Empty slice: reducing `[]int{}` with initial value `42` returns `42`
  unchanged, verifying the reducer function is never invoked.

## How to test

```
go test ./lib/... -run TestReduce -v
go build ./...
```

Both pass locally; the full `lib` package test suite (`go test ./lib/...`)
also passes unchanged.
