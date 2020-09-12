package main

type Position [3]float32

type OctNode struct {
	In        chan Position
	Center    Position
	Size      float32
	Neighbour [8]chan Position
	List      []Position
}

func displace(p Position, size float32, direction int) Position {
	var r Position
	r[0] = p[0] - size/2 + float32(direction%2)*size
	r[1] = p[1] - size/2 + float32((direction>>1)%2)*size
	r[2] = p[2] - size/2 + float32((direction>>2)%2)*size
	return r
}

func (ch *OctNode) Init() {
	for i := 0; i < len(ch.Neighbour); i++ {
		c := make(chan Position)
		ch.Neighbour[i] = c

		go func(c chan Position) {
			node := OctNode{In: c, Center: displace(ch.Center, ch.Size, i), Size: ch.Size / 2}
			for p := range c {
				node.List = append(node.List, p)
			}

		}(c)
	}

}

func main() {
	root := Oct{Center: [3]float32{0.0, 0.0, 0.0}, Size: 3000}

	/*var num int64
	num = 1000 * 1000 * 1000*/

	back := make(chan int)

	var channels [2]chan [3]float32
	for i := range channels {
		channels[i] = make(chan [3]float32)
	}

	for _, c := range channels {

		go func(ch chan [3]float32) {
			a := <-ch
			back <- 1

		}(c)
	}

	for i, c := range channels {
		c <- [3]float32{float32(i), float32(i), float32(i)}
	}

}
