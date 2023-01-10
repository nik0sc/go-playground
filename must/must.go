package must

func Must2[T1 any](p1 T1, err error) T1 {
	if err != nil {
		panic(err)
	}
	return p1
}

func Must3[T1, T2 any](p1 T1, p2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}
	return p1, p2
}

func Must4[T1, T2, T3 any](p1 T1, p2 T2, p3 T3, err error) (T1, T2, T3) {
	if err != nil {
		panic(err)
	}
	return p1, p2, p3
}
