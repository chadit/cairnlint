package modernize

import (
	"go/types"
	"sync"
)

// Parallel starts each job with the Add, go, Done shape that wg.Go replaces.
func Parallel(jobs []func()) {
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)

		go func() {
			defer wg.Done()
			job()
		}()
	}

	wg.Wait()
}

// ParallelLimited adds the whole batch at once, which is not the one-goroutine
// shape wg.Go replaces, so waitgroupgo stays quiet.
func ParallelLimited(jobs []func()) {
	var wg sync.WaitGroup

	wg.Add(len(jobs))

	for _, job := range jobs {
		go job()
	}

	wg.Wait()
}

// FieldNames walks a struct with the Len and At pair that Fields replaces.
func FieldNames(fields *types.Struct) []string {
	names := make([]string, 0, fields.NumFields())

	for i := 0; i < fields.NumFields(); i++ {
		names = append(names, fields.Field(i).Name())
	}

	return names
}

// FirstField reads one field by index, which needs no iterator.
func FirstField(fields *types.Struct) string {
	return fields.Field(0).Name()
}
