package helpers

func ArrMap[I, O any](arr []I, cb func(I) O) []O {
	r := make([]O, 0)
	for _, item := range arr {
		r = append(r, cb(item))
	}
	return r
}
