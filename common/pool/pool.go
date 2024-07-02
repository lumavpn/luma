package pool

func Get(size int) []byte {
	if size == 0 {
		return nil
	}
	return defaultAllocator.Get(size)
}

func Put(buf []byte) error {
	return defaultAllocator.Put(buf)
}
