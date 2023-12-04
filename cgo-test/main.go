package main

/*
#cgo LDFLAGS: -L./ -llibbn254.so
#include "icicle.h"
*/
import (
	"C"
	"fmt"
	"log"
	"math/big"
	"unsafe"
	goicicle "github.com/ingonyama-zk/icicle/goicicle"
	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"

	icicle "github.com/ingonyama-zk/icicle/goicicle/curves/bn254"
)

type custom [32]byte

func extra() {
	num := 3
	_, _, g1Aff, _ := bn254.Generators()
	pointsBytesA := fp.Bytes * 2 * num
	points := make([]bn254.G1Affine, num)
	points[0] = g1Aff
	points[1].Add(&g1Aff, &g1Aff)
	points[2].Add(&points[1], &points[1])

	fmt.Println(points[2].String())

	copyADone := make(chan unsafe.Pointer, num)
	go CopyPointsToDevice(points, pointsBytesA, copyADone)
	ptr := <-copyADone

	out := make([]icicle.G1PointAffine, num)
	goicicle.CudaMemCpyDtoH[icicle.G1PointAffine](out, ptr, pointsBytesA)
	outAff := AffineToGnarkAffine(&out[2])
	fmt.Println(outAff.String())

	FreeDevicePointer(ptr)

	wireValuesB := make([]fr.Element, num)
	for i := 0; i < num; i++ {
		wireValuesB[i].SetUint64(uint64(i + 10+256))
	}

	var regInt big.Int
	fmt.Println("Regular", wireValuesB[2].BigInt(&regInt))

	scalarBytes := len(wireValuesB) * fr.Bytes
	// Copy scalars to the device and retain ptr to them
	copyDone := make(chan unsafe.Pointer, 1)
	go CopyToDevice(wireValuesB, scalarBytes, copyDone)
	frptr := <-copyDone

	fmt.Println("fr.Bytes", fr.Bytes)
	outIciFr := make([]byte, num*32)
	goicicle.CudaMemCpyDtoH[byte](outIciFr, frptr, fr.Bytes*num)
	//outFr := ScalarToGnarkFr(&outIciFr[2])
	fmt.Println("fr ", outIciFr)


	sum, _, err := MsmOnDevice(ptr, frptr, num, true)
	if err != nil {
		fmt.Printf("err %v\n", err)
	}
	fmt.Println("sum", sum.String())




}

func MsmOnDevice(scalars_d, points_d unsafe.Pointer, count int, convert bool) (bn254.G1Jac, unsafe.Pointer, error) {
	pointBytes := fp.Bytes * 3  // 3 Elements because of 3 coordinates
	out_d, _ := goicicle.CudaMalloc(pointBytes)

	icicle.Commit(out_d, scalars_d, points_d, count, 10)

	if convert {
	outHost := make([]icicle.G1ProjectivePoint, 1)
	goicicle.CudaMemCpyDtoH[icicle.G1ProjectivePoint](outHost, out_d, pointBytes)

	return *G1ProjectivePointToGnarkJac(&outHost[0]), nil, nil
	}

	return bn254.G1Jac{}, out_d, nil
}

func G1ProjectivePointToGnarkJac(p *icicle.G1ProjectivePoint) *bn254.G1Jac {
	var p1 bn254.G1Jac
	p1.FromAffine(ProjectiveToGnarkAffine(p))

	return &p1
}

func ScalarToGnarkFr(f *icicle.G1ScalarField) *fr.Element {
	fb := f.ToBytesLe()
	var b32 [32]byte
	copy(b32[:], fb[:32])

	v, e := fr.LittleEndian.Element(&b32)

	if e != nil {
	panic(fmt.Sprintf("unable to create convert point %v got error %v", f, e))
	}

	return &v
}

func AffineToGnarkAffine(p *icicle.G1PointAffine) *bn254.G1Affine {
	return ProjectiveToGnarkAffine(p.ToProjective())
}

func BaseFieldToGnarkFp(f *icicle.G1BaseField) *fp.Element {
	fb := f.ToBytesLe()
	var b32 [32]byte
	copy(b32[:], fb[:32])

	v, e := fp.LittleEndian.Element(&b32)

	if e != nil {
		panic(fmt.Sprintf("unable to convert point %v got error %v", f, e))
	}

	return &v
}


func ProjectiveToGnarkAffine(p *icicle.G1ProjectivePoint) *bn254.G1Affine {
	px := BaseFieldToGnarkFp(&p.X)
	py := BaseFieldToGnarkFp(&p.Y)
	pz := BaseFieldToGnarkFp(&p.Z)

	zInv := new(fp.Element)
	x := new(fp.Element)
	y := new(fp.Element)

	zInv.Inverse(pz)

	x.Mul(px, zInv)
	y.Mul(py, zInv)

	return &bn254.G1Affine{X: *x, Y: *y}
}

func BatchConvertFromG1Affine(elements []bn254.G1Affine) []icicle.G1PointAffine {
	var newElements []icicle.G1PointAffine
	for _, e := range elements {
		var newElement icicle.G1ProjectivePoint
		FromG1AffineGnark(&e, &newElement)

		newElements = append(newElements, *newElement.StripZ())
	}
	return newElements
}

func NewFieldFromFpGnark[T icicle.G1BaseField | icicle.G1ScalarField](element fp.Element) *T {
	s := icicle.ConvertUint64ArrToUint32Arr(element.Bits()) // get non-montgomry
	return &T{S: s}
}

func FromG1AffineGnark(gnark *bn254.G1Affine, p *icicle.G1ProjectivePoint) *icicle.G1ProjectivePoint {
	var z icicle.G1BaseField
	z.SetOne()

	p.X = *NewFieldFromFpGnark[icicle.G1BaseField](gnark.X)
	p.Y = *NewFieldFromFpGnark[icicle.G1BaseField](gnark.Y)
	p.Z = z

	return p
}

func CopyToDevice(scalars []fr.Element, bytes int, copyDone chan unsafe.Pointer) {
	devicePtr, _ := goicicle.CudaMalloc(bytes)
	goicicle.CudaMemCpyHtoD[fr.Element](devicePtr, scalars, bytes)
	MontConvOnDevice(devicePtr, len(scalars), false)

	copyDone <- devicePtr
}

func MontConvOnDevice(scalars_d unsafe.Pointer, size int, is_into bool) {
	if is_into {
	icicle.ToMontgomery(scalars_d, size)
	} else {
	icicle.FromMontgomery(scalars_d, size)
	}
}

func CopyPointsToDevice(points []bn254.G1Affine, pointsBytes int, copyDone chan unsafe.Pointer) {
	if pointsBytes == 0 {
		copyDone <- nil
	} else {
		devicePtr, _ := goicicle.CudaMalloc(pointsBytes)
		iciclePoints := BatchConvertFromG1Affine(points)
		goicicle.CudaMemCpyHtoD[icicle.G1PointAffine](devicePtr, iciclePoints, pointsBytes)

		copyDone <- devicePtr
	}
}

func FreeDevicePointer(ptr unsafe.Pointer) {
	goicicle.CudaFree(ptr)
}

func main() {
	a := make([]C.float, 4)
	b := make([]C.float, 4)
	d := make([]C.float, 4)

	for i := 0 ; i<4 ; i++ {
		a[i] = C.float(i)
		b[i] = C.float(i)
	}

	d_a, err := goicicle.CudaMalloc(4*len(a))
	if err != nil {
		log.Fatalf("%v", err)
	}


	ret := goicicle.CudaMemCpyHtoD[C.float](d_a, a, 4*len(a))
	if ret != 0 {
		log.Fatal("ret not 0")
	}

	goicicle.CudaMemCpyDtoH[C.float](d, d_a, 4*len(a))
	fmt.Println("d", d[3])


	goicicle.CudaFree(d_a)

	//d_c, err := CudaMalloc(4*len(c))
	//if err != nil {
	//	log.Fatalf("%v", err)
	//}

	extra()


}

