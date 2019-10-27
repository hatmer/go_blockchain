package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

const BLOCK_SIZE = 2 //1024
const BUFFER_SIZE = 4096
const PORT = 8080

var blockNumber = 0

type entry []byte

type block struct {
	blockNumber    int
	blockHash      string
	hashPrevBlock  string
	hashMerkleRoot string
	time           int64
	entries        []entry
}

type dataPasser struct {
	ch chan entry
}

/*
 * write block to disk
 */
func (b *block) write() error {
	// TODO
	return errors.New("could not write block\n")
}

/*func sha(x []byte) [32]byte {
	return sha256.Sum256(sha256.Sum256(x))
}*/
func sha(x []byte) []byte {
	once := sha256.Sum256(x)
	s := once[:]
	twice := sha256.Sum256(s)
	t := twice[:]
	return t
}

/**************************/

/*
 * calculates merkle root
 */

/*
 * Algorithm:
 *
 * put_index = 0
 * get_index = 0
 *
 * len_this_round = block_size
 * while len_this_round > 1:
 *   while put_index < (len_this_round / 2):
 *     slice[put_index] = sha(slice[get_index] + slice[get_index+1])
 *     put_index += 1
 *     get_index += 2
 *   len_this_round /= 2
 *
 */

func getMerkleRoot(entries []entry) (string, error) {
	if len(entries) != BLOCK_SIZE {
		return "", errors.New("incorrect block size\n")
	}

	// hash each entry
	var hashes []entry
	for _, x := range entries {
		hash := sha([]byte(x))
		hashes = append(hashes, hash)
	}

	// calculate Merkle Root in place
	get_index := 0
	for len_this_round := BLOCK_SIZE; len_this_round > 1; len_this_round /= 2 {
		for put_index := 0; put_index < (len_this_round / 2); put_index += 1 {
			to_hash := append([]byte(entries[get_index]), []byte(entries[get_index+1])...)
			hashes[put_index] = sha(to_hash)
			get_index += 2
		}
	}

	fmt.Printf("merkle root is %x\n", hashes[0])
	return string(hashes[0]), nil
}

/*
 * generates block
 */
func generate(ch chan entry, hashPrev string) {
	entries := make([]entry, 0, BLOCK_SIZE)
	fmt.Printf("block number: %d\n", blockNumber)
	for i := 0; i < BLOCK_SIZE; i += 1 {
		val := <-ch
		entries = append(entries, val)
		fmt.Printf("entries: %x\n", entries)
	}
	t := time.Now().Unix()
	merkleRoot, err := getMerkleRoot(entries)
	if err == nil {
		blockHash := string(sha([]byte(hashPrev + merkleRoot + string(t))))
		fmt.Printf("blockHash: %x\n", blockHash)
		b := block{blockNumber, blockHash, hashPrev, merkleRoot, t, entries}
		fmt.Printf("block: %x\n", b)
		blockNumber += 1
		generate(ch, blockHash)
	} else {
		fmt.Printf("could not generate block. crashing...\n")
	}
}

/********************************/

/*
 * handles http request, writes to channel, and writes response (status and block number)
 */
func (p *dataPasser) handler(w http.ResponseWriter, r *http.Request) {
	input := []byte(r.URL.Path[1:])
	fmt.Printf("input is %s\n", input)
	p.ch <- input
	fmt.Fprintf(w, "done, %d\n", blockNumber)
}

/*
 * runs server on localhost 8080
 */
func main() {
	secret := string(sha([]byte("secret")))
	ch := make(chan entry, BUFFER_SIZE)

	/* run block builder */
	go generate(ch, secret)

	/* run server */
	p := dataPasser{ch}
	http.HandleFunc("/", p.handler)
	port := ":" + strconv.Itoa(PORT)
	log.Fatal(http.ListenAndServe(port, nil))
}
