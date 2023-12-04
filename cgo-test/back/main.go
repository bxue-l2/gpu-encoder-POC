package main

/*
#cgo CFLAGS: -I /usr/local/cuda/include
#cgo LDFLAGS: -L/usr/local/cuda/lib64 -lcudart
#include <cuda.h>
#include <cuda_runtime.h>
*/
import "C"
import "fmt"
import "log"
import "errors"
import "unsafe"


func main() {
	a := make([]C.float, 4)
	b := make([]C.float, 4)
	d := make([]C.float, 4)

	for i := 0 ; i<4 ; i++ {
		a[i] = C.float(i)
		b[i] = C.float(i)
	}

	d_a, err := CudaMalloc(4*len(a))
	if err != nil {
		log.Fatalf("%v", err)
	}


	ret := CudaMemCpyHtoD[C.float](d_a, a, 4*len(a))
	if ret != 0 {
		log.Fatal("ret not 0")
	}

	CudaMemCpyDtoH[C.float](d, d_a, 4*len(a))
	fmt.Println("d", d[3])


	CudaFree(d_a)

	//d_c, err := CudaMalloc(4*len(c))
	//if err != nil {
	//	log.Fatalf("%v", err)
	//}




}

func CudaMalloc(size int) (dp unsafe.Pointer, err error) {
	var p C.void
	dp = unsafe.Pointer(&p)
	if err := C.cudaMalloc(&dp, C.size_t(size)); err != 0 {
		return nil, errors.New("could not create memory space")
	}
	return dp, nil
}

func CudaMemCpyHtoD[T any](dst_d unsafe.Pointer, src []T, size int) int {
	src_c := unsafe.Pointer(&src[0])
	if err := C.cudaMemcpy(dst_d, src_c, C.size_t(size), 1); err != 0 {
		return -1
	}
	return 0
}

func CudaFree(dp unsafe.Pointer) int {
	if err := C.cudaFree(dp); err != 0 {
		return -1
	}
	return 0
}

func CudaMemCpyDtoH[T any](dst []T, src_d unsafe.Pointer, size int) int {
	dst_c := unsafe.Pointer(&dst[0])

	if err := C.cudaMemcpy(dst_c, src_d, C.size_t(size), 2); err != 0 {
		return -1
	}
	return 0
}
