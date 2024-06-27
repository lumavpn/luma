package pool

func Map[T any, N any](arr []T, block func(it T) N) []N {
	retArr := make([]N, 0, len(arr))
	for index := range arr {
		retArr = append(retArr, block(arr[index]))
	}
	return retArr
}

func LenMulti(buffers []*Buffer) int {
	var n int
	for _, buffer := range buffers {
		n += buffer.Len()
	}
	return n
}

func ToSliceMulti(buffers []*Buffer) [][]byte {
	return Map(buffers, func(it *Buffer) []byte {
		return it.Bytes()
	})
}

func CopyMulti(toBuffer []byte, buffers []*Buffer) int {
	var n int
	for _, buffer := range buffers {
		n += copy(toBuffer[n:], buffer.Bytes())
	}
	return n
}

func (b *Buffer) Release() {
	if b == nil || !b.managed {
		return
	}
	if b.refs.Load() > 0 {
		return
	}
	Put(b.data)
	*b = Buffer{}
}

func ReleaseMulti(buffers []*Buffer) {
	for _, buffer := range buffers {
		buffer.Release()
	}
}
