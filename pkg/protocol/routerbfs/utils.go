package routerbfs

func sum(v []uint32) (n uint32) {
	for i := 0; i < len(v); i++ {
		n += v[i]
	}
	return
}
