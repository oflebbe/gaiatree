package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

type Position [5]float32

type OctNode struct {
	count int32
}

const NUM = 1024.0
const INUM = 1024
const SIZE = 40000.0

func pos2ind(x float32) (xx int, e error) {
	xx = int(NUM * (x/SIZE + 0.5))

	if xx < 0 || xx >= int(NUM) {
		e = errors.New("out of range")
	}
	return
}

func position2index(p Position) (index int, err error) {
	xx, errX := pos2ind(p[2])
	yy, errY := pos2ind(p[3])
	zz, errZ := pos2ind(p[4])
	if errX != nil || errY != nil || errZ != nil {
		err = errors.New("out of range")
	} else {
		index = xx*(INUM*INUM) + yy*(INUM) + zz
	}
	return
}

func oneBlock(oct []int32, positions []Position) {

	for _, pos := range positions {

		i, err := position2index(pos)
		if err == nil {
			atomic.AddInt32(&oct[i], 1)

		}
	}
	return
}

// HandleBlocks read filenames from chanel ch and puts them on chanel output
func HandleBlocks(oct []int32, ch chan []Position, wg *sync.WaitGroup) {
	defer wg.Done()
	for s := range ch {
		oneBlock(oct, s)
		fmt.Printf("%s\n", s)
	}
	fmt.Printf("End of Jobs\n")
}

func main() {
	oct := make([]int32, INUM*INUM*INUM)
	input, err := os.Open("result.dat")
	if err != nil {
		fmt.Errorf("Cannot open result")
	}

	// Reader
	for i := 0; i < runtime.NumCPU(); i++ {
		go func(wg *sync.WaitGroup) {
			wg.Add(1)
			OneBlock(oct, positions)
		}(&wg)
	}

	for {
		positions := make([]Position, 1024*20)
		err := binary.Read(input, binary.LittleEndian, &positions)
		if err != nil {
			fmt.Errorf("Readerror")
		}

		/*	byteBuf := make([]byte, 20*1024)
			remainingLen := 0

			len, err := input.Read(byteBuf[remainingLen:])
			if err != nil {
				fmt.Errorf("Error")
			}
			usableLen := 20 * (len / 20)
			remainingLen = len - usableLen
			binary.Read( 	)*/

	}
}
