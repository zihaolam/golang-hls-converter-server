package utils

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

func Parallelize[TArg any, TRes any](fn func(arg TArg) (TRes, error), args ...TArg) ([]TRes, []error) {
	var wg sync.WaitGroup
	n := len(args)
	resCh := make(chan TRes, n)
	errCh := make(chan error, n)
	for _, arg := range args {
		wg.Add(1)
		go func(arg TArg) {
			defer wg.Done()
			r, err := fn(arg)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- r
		}(arg)
	}
	go func() {
		wg.Wait()
		close(resCh)
		close(errCh)
	}()

	res := []TRes{}
	err := []error{}

	for r := range resCh {
		res = append(res, r)
	}
	for e := range errCh {
		err = append(err, e)
	}

	return res, err
}

func ChunkArray[T any](arr []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(arr); i += chunkSize {
		end := i + chunkSize
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

func Flatten[T any](nested [][]T) []T {
	flattened := make([]T, 0)

	for _, i := range nested {
		flattened = append(flattened, i...)
	}

	return flattened
}

var _validator *validator.Validate

func Validate(obj interface{}) error {
	_validator = validator.New()
	return _validator.Struct(obj)
}
