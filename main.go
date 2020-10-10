package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// OctNode s
type OctNode struct {
	count int32
}

// NUM n
const NUM = 100.0

// INUM i
const INUM = 100

// SIZE s
const SIZE = 40000.0

// Coord Of the stars
type Coord struct {
	Ra  float32
	Dec float32
	X   float32
	Y   float32
	Z   float32
}

func pos2ind(x float32) (xx int, e error) {
	xx = int(NUM * (x/SIZE + 0.5))

	if xx < 0 || xx >= int(NUM) {
		e = errors.New("out of range")
	}
	return
}

func position2index(p Coord) (index int, err error) {
	xx, errX := pos2ind(p.X)
	yy, errY := pos2ind(p.Y)
	zz, errZ := pos2ind(p.Z)
	if errX != nil || errY != nil || errZ != nil {
		err = errors.New("out of range")
	} else {
		index = xx*(INUM*INUM) + yy*(INUM) + zz
	}
	return
}

func oneBlock(oct []int32, positions []Coord) {

	for _, pos := range positions {

		i, err := position2index(pos)
		if err == nil {
			atomic.AddInt32(&oct[i], 1)
		}
	}
	return
}

/*

// HandleBlocks read filenames from chanel ch and puts them on chanel output
func HandleBlocks(oct []int32, ch chan []Position, wg *sync.WaitGroup) {
	defer wg.Done()
	for s := range ch {
		oneBlock(oct, s)
		fmt.Printf("%s\n", s)
	}
	fmt.Printf("End of Jobs\n")
}*/

// ReadOneBlock reads on file
func readOneBlock(in []byte) (result []Coord) {
	uncompressor, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		log.Fatalf("Could not uncompress")
	}
	scanner := bufio.NewScanner(uncompressor)
	scanner.Scan() // skip first line

	for scanner.Scan() {
		line := scanner.Text()
		toks := strings.Split(line, ",")
		ra, err := strconv.ParseFloat(toks[5], 32)
		if err != nil {
			log.Fatalf("line %s not parsable: ra", line)
		}

		dec, err := strconv.ParseFloat(toks[7], 32)
		if err != nil {
			log.Fatalf("line %s not parsable: dec", line)
		}

		if toks[9] == "" {
			continue
		}
		parallax, err := strconv.ParseFloat(toks[9], 32)
		if err != nil {
			log.Fatalf("line %s not parsable parallax %s", line, toks[9])
		}
		if parallax < 0. {
			// https://astronomy.stackexchange.com/questions/26250/what-is-the-proper-interpretation-of-a-negative-parallax
			continue
		}
		sra, cra := math.Sincos(ra * math.Pi / 180.0)
		sdec, cdec := math.Sincos(dec * math.Pi / 180.0)
		r := 1.58125074e-5 / (parallax / (1000 * 3600) * math.Pi / 180.)
		x := r * cra * cdec
		y := r * sra * cdec
		z := r * sdec
		c := Coord{Ra: float32(ra), Dec: float32(dec), X: float32(x), Y: float32(y), Z: float32(z)}
		// println(x, y, z)

		result = append(result, c)
	}
	return
}

// Handles the complete tar
func handleTar(fn string, fileChannel chan []byte) {
	// open Tar

	file, err := os.Open(fn)
	if err != nil {
		log.Fatalf("couldn't open %\n")
		os.Exit(1)
	}
	tarReader := tar.NewReader(file)
	counter := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if header.FileInfo().IsDir() {
			continue
		}
		if !strings.HasSuffix(header.Name, "csv.gz") {
			continue
		}
		fmt.Printf("Reading %s\n", header.Name)

		buf, err := ioutil.ReadAll(tarReader)
		if err != nil {
			//log.Fatal(err)
			return
		}
		fileChannel <- buf
		counter++
		if counter > 500 {
			return
		}
	}

}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	oct := make([]int32, INUM*INUM*INUM)

	fileChannel := make(chan []byte)
	coordChannel := make(chan []Coord)

	countReadChannel := make(chan int)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for in := range fileChannel {
				coordChannel <- readOneBlock(in)
			}
			countReadChannel <- 1 // done
		}()
	}

	countProcessChannel := make(chan int)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for in := range coordChannel {
				oneBlock(oct, in)
			}
			countProcessChannel <- 1 // done
		}()
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)

			nonzero := 0
			sumnonzero := int32(0)

			for i := 0; i < len(oct); i++ {
				if oct[i] != 0 {
					nonzero++
					sumnonzero += oct[i]
				}
			}
			fmt.Printf("%d %f %f\n", nonzero, float32(sumnonzero)/float32(nonzero), float32(sumnonzero)/float32(len(oct)))
		}

	}()

	handleTar("gaia.tar", fileChannel)
	close(fileChannel)
	for i := 0; i < runtime.NumCPU(); i++ {
		<-countReadChannel
	}
	close(countReadChannel)
	close(coordChannel)

	for i := 0; i < runtime.NumCPU(); i++ {
		<-countProcessChannel
	}
	close(countProcessChannel)

	result, err := os.Create("result.dat")
	if err != nil {
		fmt.Errorf("Cannot open result")
	}

	binary.Write(result, binary.LittleEndian, oct[:])
	result.Close()
}
