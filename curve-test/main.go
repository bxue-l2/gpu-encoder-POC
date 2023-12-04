package main

/*
#cgo LDFLAGS: -L./ -llibbn254.so
#include "icicle.h"
*/
import (
	"C"

	goicicle "github.com/ingonyama-zk/icicle/goicicle"
)
import (
	"fmt"
	"unsafe"

	ig "github.com/bx3/curve-test/bn254"
	curve "github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

/*
func testFr() {
	a := make([]fr.Element, 32)
	b := make([]fr.Element, 32)
	c := make([]fr.Element, 32)

	for i := 0; i < 32; i++ {
		a[i].SetUint64(uint64(i))
		b[i].SetUint64(uint64(i + 100))
		c[i].SetUint64(uint64(i + 1000))
	}

	n := len(a)

	sizeBytes := n * fr.Bytes

	copyADone := make(chan unsafe.Pointer, 1)
	copyBDone := make(chan unsafe.Pointer, 1)
	copyCDone := make(chan unsafe.Pointer, 1)

	go CopyToDevice(a, sizeBytes, copyADone)
	go CopyToDevice(b, sizeBytes, copyBDone)
	go CopyToDevice(c, sizeBytes, copyCDone)

	a_device := <-copyADone
	b_device := <-copyBDone
	c_device := <-copyCDone

	computeInttNttDone := make(chan error, 1)
	computeInttNttOnDevice := func(devicePointer unsafe.Pointer) {
		scalarsInterp := icicle.Msm()
		computeInttNttDone <- nil
	}

	go computeInttNttOnDevice(a_device)
	go computeInttNttOnDevice(b_device)
	go computeInttNttOnDevice(c_device)
	_, _, _ = <-computeInttNttDone, <-computeInttNttDone, <-computeInttNttDone

	PolyOps(a_device, b_device, c_device, pk.DenDevice, n)
}

DST := []byte("-")
	msg := []byte("hi")
	a, err := curve.HashToG1(msg, DST)
	if err != nil {
		fmt.Println(err)
		return
	}
	var b curve.G1Affine
	b.Add(&a, &a)

	g1List := make([]curve.G1Affine, 4)
	g1List[0] = a
	g1List[1] = b
	g1List[2] = b
	g1List[3] = b

	bn254.G1

	var one icicle.G1ScalarField
	one.SetOne()
	fmt.Printf("icicle G1ScalarField %v\n", one.ToBytesLe())

	frList := make([]fr.Element, 2)
	frList[0].SetUint64(uint64(1))

	fmt.Printf("gnark G1ScalarField %v\n", frList[0].String())
	//frList[1].SetUint64(uint64(2))
	//bn254.Msm(g1List, frList, 0)

*/

func main() {
	num := 2

	wireValuesB := make([]fr.Element, num)
	for i := 0; i < num; i++ {
		wireValuesB[i].SetUint64(uint64(i + 10))
	}
	wireValuesBSize := len(wireValuesB)
	scalarBytes := wireValuesBSize * fr.Bytes
	// Copy scalars to the device and retain ptr to them
	copyDone := make(chan unsafe.Pointer, 1)
	ig.CopyToDevice(wireValuesB, scalarBytes, copyDone)
	wireValuesBDevicePtr := <-copyDone

	wireValuesBDevice := ig.OnDeviceData{
			P:    wireValuesBDevicePtr,
			Size: wireValuesBSize,
		}

	_, _, g1Aff, _ := curve.Generators()
	points := make([]curve.G1Affine, num)
	points[0] =  g1Aff
	fmt.Printf("%v point %v\n", 0, points[0].String())
	for i := 1; i < num; i++ {
		points[i].Add(&points[i-1], &points[i-1])
		fmt.Printf("%v point %v\n", i, points[i].String())
	}

	pointsBytesA := num * fp.Bytes * 2
	copyADone := make(chan unsafe.Pointer, 1)

	go ig.CopyPointsToDevice(points, pointsBytesA, copyADone) // Make a function for points
	fmt.Println("CopyPointsToDevice field done")

	point_pointer := <-copyADone

	sum, _, err := ig.MsmOnDevice(wireValuesBDevice.P, point_pointer, wireValuesBDevice.Size, true)
	if err != nil {
		fmt.Printf("err %v\n", err)
	}
	var iciSum curve.G1Affine
	iciSum.FromJacobian(&sum)
	fmt.Println("sum", sum.String())

	var gnarkSum curve.G1Affine
	gnarkSum.MultiExp(points, wireValuesB, ecc.MultiExpConfig{})
	fmt.Println("gnarkSum", gnarkSum.String())

	/*
		num := 4
		a := make([]fr.Element, num)
		b := make([]fr.Element, num)
		c := make([]fr.Element, num)

		for i := 0; i < num; i++ {
			a[i].SetUint64(uint64(i))
			b[i].SetUint64(uint64(i + 100))
			c[i].SetUint64(uint64(i + 1000))
		}

		aI := ig.BatchConvertFromFrGnark[bn254.G1ScalarField](a)
		for i := 0; i < num; i++ {
			fmt.Printf("%v ", a[i].String())
		}
		fmt.Println()

		bn254.Ntt(&aI, false, 0)

		aBack := ig.BatchConvertG1ScalarFieldToFrGnark(aI)
		for i := 0; i < num; i++ {
			fmt.Printf("%v ", aBack[i].String())
		}
		fmt.Println()
	*/
	//b := ig.NewFieldFromFrGnark[bn254.G1ScalarField](a)
	//fmt.Printf("%v\n", b.ToBytesLe())

}

func FreeDevicePointer(ptr unsafe.Pointer) {
        goicicle.CudaFree(ptr)
}
